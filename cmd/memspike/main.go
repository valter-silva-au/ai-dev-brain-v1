// memspike is a throwaway decision-gate program for the Stage 3 vector
// memory port into adb (see .wiki/decisions/0002-ruflo-dispatch-and-
// vector-memory-in-adb.md on the consumer monorepo). It verifies two
// things across Windows / Linux / macOS:
//
//  1. `modernc.org/sqlite` (pure-Go SQLite driver, no cgo) opens + reads
//     + writes a database at a Windows temp path without pain.
//  2. `github.com/coder/hnsw` builds, accepts 100 × 384-dim vectors, and
//     answers a nearest-neighbour query in reasonable time.
//
// Notes on library selection during the spike:
//
//   - `coder/hnsw` v0.6.1 has a compile-time break (imports v1 of
//     `github.com/google/renameio` but calls the removed `TempFile`).
//     Downgraded to v0.2.0 which builds clean on all three OSes.
//   - v0.2.0's API uses an `Embeddable` interface (ID()+Embedding())
//     rather than v0.6.x's `MakeNode` free function. Slightly more
//     boilerplate for callers; stable enough for the real port.
//
// If this runs green on all three OSes, Stage 3.1 proceeds with the
// real `internal/memory/` package. If it hits cgo / build / runtime
// walls, we pivot to a bridge against ruflo's AgentDB.
//
// This file is not intended to ship. Delete after Stage 3 merges.
package main

import (
	"database/sql"
	"fmt"
	mathrand "math/rand"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/coder/hnsw"
	_ "modernc.org/sqlite"
)

// mathrandLegacy returns a math/rand v1 *Rand (what coder/hnsw v0.2.0 expects
// for its Rng field) seeded deterministically.
func mathrandLegacy(seed int64) *mathrand.Rand {
	return mathrand.New(mathrand.NewSource(seed))
}

const (
	dim      = 384
	numVec   = 100
	topK     = 5
	dbFile   = "memspike.sqlite"
	hnswM    = 16
	efSearch = 200
)

// spikeEmbeddable satisfies hnsw.Embeddable for a fixed-id + vector pair.
type spikeEmbeddable struct {
	id  string
	vec []float32
}

func (s spikeEmbeddable) ID() string             { return s.id }
func (s spikeEmbeddable) Embedding() []float32   { return s.vec }

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "memspike failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	tmp, err := os.MkdirTemp("", "memspike-*")
	if err != nil {
		return fmt.Errorf("mktemp: %w", err)
	}
	defer os.RemoveAll(tmp)
	dbPath := filepath.Join(tmp, dbFile)
	fmt.Printf("workspace=%s\n", tmp)

	if err := sqliteRoundtrip(dbPath); err != nil {
		return fmt.Errorf("sqlite roundtrip: %w", err)
	}

	if err := hnswRoundtrip(); err != nil {
		return fmt.Errorf("hnsw roundtrip: %w", err)
	}

	fmt.Println("\nmemspike: all probes green")
	return nil
}

func sqliteRoundtrip(dbPath string) error {
	fmt.Printf("\n[1/2] modernc.org/sqlite probe\n")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec(`create table kv (k text primary key, v text)`); err != nil {
		return fmt.Errorf("create: %w", err)
	}
	if _, err := db.Exec(`insert into kv(k, v) values (?, ?)`, "hello", "world"); err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	var got string
	if err := db.QueryRow(`select v from kv where k = ?`, "hello").Scan(&got); err != nil {
		return fmt.Errorf("select: %w", err)
	}
	if got != "world" {
		return fmt.Errorf("roundtrip mismatch: got %q want %q", got, "world")
	}
	if st, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("stat db file: %w", err)
	} else {
		fmt.Printf("  ok: %s (%d bytes)\n", st.Name(), st.Size())
	}
	return nil
}

func hnswRoundtrip() error {
	fmt.Printf("\n[2/2] github.com/coder/hnsw probe: insert %d x %d-dim, query k=%d\n", numVec, dim, topK)

	// Deterministic seed so runs are reproducible across OSes.
	rng := rand.New(rand.NewPCG(42, 1337))

	g := hnsw.NewGraph[spikeEmbeddable]()
	g.M = hnswM
	g.EfSearch = efSearch
	// Deterministic RNG so test behaviour is reproducible across OSes.
	g.Rng = mathrandLegacy(42)

	// Insert numVec random vectors with deterministic ids.
	startInsert := time.Now()
	items := make([]spikeEmbeddable, numVec)
	for i := 0; i < numVec; i++ {
		v := make([]float32, dim)
		for j := 0; j < dim; j++ {
			v[j] = float32(rng.Float64()*2 - 1)
		}
		items[i] = spikeEmbeddable{id: strconv.Itoa(i), vec: v}
	}
	g.Add(items...)
	insertDur := time.Since(startInsert)
	fmt.Printf("  insert: %d vectors in %s (%.2f/sec)\n",
		numVec, insertDur, float64(numVec)/insertDur.Seconds())

	// Query 1: self-query — top hit MUST be vec 0.
	startSearch := time.Now()
	hits := g.Search(items[0].vec, topK)
	searchDur := time.Since(startSearch)
	if len(hits) == 0 {
		return fmt.Errorf("search returned 0 hits")
	}
	fmt.Printf("  self-query: %d hits in %s\n", len(hits), searchDur)
	for i, h := range hits {
		fmt.Printf("    [%d] id=%s\n", i, h.ID())
	}
	if hits[0].ID() != "0" {
		return fmt.Errorf("self-query: expected top hit id=0, got id=%s (HNSW is probabilistic; use deterministic Rng in tests)", hits[0].ID())
	}

	// Query 2: a novel vector never in the set. Should return k hits,
	// none of which are identity matches (score < 1.0 equivalent).
	query := make([]float32, dim)
	for j := 0; j < dim; j++ {
		query[j] = float32(rng.Float64()*2 - 1)
	}
	hits = g.Search(query, topK)
	if len(hits) != topK {
		return fmt.Errorf("novel-query: expected %d hits, got %d", topK, len(hits))
	}
	fmt.Printf("  novel-query: %d hits (no exact match expected)\n", len(hits))

	return nil
}

//go:build !windows

package core

import (
	"os"

	"golang.org/x/sys/unix"
)

// lockFile acquires an advisory exclusive lock on the given file using flock(2).
// Returns a release function that unlocks the file. Blocks until the lock is
// available. On Unix this uses BSD-style flock, which is released automatically
// when the file descriptor is closed, so the release function is idempotent.
func lockFile(f *os.File) (func(), error) {
	fd := int(f.Fd())
	if err := unix.Flock(fd, unix.LOCK_EX); err != nil {
		return func() {}, err
	}
	return func() { _ = unix.Flock(fd, unix.LOCK_UN) }, nil
}

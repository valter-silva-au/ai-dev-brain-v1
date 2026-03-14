package hive

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
	"gopkg.in/yaml.v3"
)

// MessageBus defines the interface for the Hive Mind messaging system.
type MessageBus interface {
	Publish(msg models.HiveMessage) error
	Subscribe(recipient string) ([]models.HiveMessage, error)
	MarkProcessed(messageID string) error
	GetConversation(conversationID string) ([]models.HiveMessage, error)
}

// messageBusStore is the file-based implementation of MessageBus.
type messageBusStore struct {
	basePath string
	counter  int
}

// NewMessageBus creates a new MessageBus instance.
func NewMessageBus(basePath string) MessageBus {
	return &messageBusStore{
		basePath: basePath,
		counter:  0,
	}
}

// Publish publishes a message to the recipient's inbox.
func (m *messageBusStore) Publish(msg models.HiveMessage) error {
	// Generate message ID if not provided
	if msg.ID == "" {
		id, err := m.generateMessageID()
		if err != nil {
			return fmt.Errorf("failed to generate message ID: %w", err)
		}
		msg.ID = id
	}

	// Set default values
	if msg.Date == "" {
		msg.Date = time.Now().UTC().Format(time.RFC3339)
	}
	if msg.Status == "" {
		msg.Status = models.HiveMessagePending
	}

	// Write message to inbox
	inboxPath := filepath.Join(m.basePath, "channels", "inbox", msg.To)
	if err := os.MkdirAll(inboxPath, 0o755); err != nil {
		return fmt.Errorf("failed to create inbox directory: %w", err)
	}

	msgPath := filepath.Join(inboxPath, fmt.Sprintf("%s.yaml", msg.ID))
	if err := m.writeMessage(msgPath, msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// Subscribe retrieves all messages for a recipient from their inbox.
func (m *messageBusStore) Subscribe(recipient string) ([]models.HiveMessage, error) {
	inboxPath := filepath.Join(m.basePath, "channels", "inbox", recipient)

	// Check if inbox exists
	if _, err := os.Stat(inboxPath); os.IsNotExist(err) {
		return []models.HiveMessage{}, nil
	}

	// Read all .yaml files from inbox
	entries, err := os.ReadDir(inboxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read inbox directory: %w", err)
	}

	var messages []models.HiveMessage
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		msgPath := filepath.Join(inboxPath, entry.Name())
		msg, err := m.readMessage(msgPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read message %s: %w", entry.Name(), err)
		}
		messages = append(messages, msg)
	}

	// Sort by date
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Date < messages[j].Date
	})

	return messages, nil
}

// MarkProcessed moves a message from inbox to archive.
func (m *messageBusStore) MarkProcessed(messageID string) error {
	// Find message in any inbox
	msgPath, err := m.findMessage(messageID)
	if err != nil {
		return fmt.Errorf("failed to find message: %w", err)
	}

	// Read the message
	msg, err := m.readMessage(msgPath)
	if err != nil {
		return fmt.Errorf("failed to read message: %w", err)
	}

	// Parse date to get year and month
	parsedTime, err := time.Parse(time.RFC3339, msg.Date)
	if err != nil {
		return fmt.Errorf("failed to parse message date: %w", err)
	}

	year := parsedTime.Format("2006")
	month := parsedTime.Format("01")

	// Create archive path
	archivePath := filepath.Join(m.basePath, "channels", "archive", year, month)
	if err := os.MkdirAll(archivePath, 0o755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Move message to archive
	archiveFile := filepath.Join(archivePath, fmt.Sprintf("%s.yaml", msg.ID))
	msg.Status = models.HiveMessageArchived

	if err := m.writeMessage(archiveFile, msg); err != nil {
		return fmt.Errorf("failed to write archived message: %w", err)
	}

	// Remove from inbox
	if err := os.Remove(msgPath); err != nil {
		return fmt.Errorf("failed to remove message from inbox: %w", err)
	}

	return nil
}

// GetConversation retrieves all messages for a specific conversation.
func (m *messageBusStore) GetConversation(conversationID string) ([]models.HiveMessage, error) {
	var messages []models.HiveMessage

	// Scan inbox
	inboxRoot := filepath.Join(m.basePath, "channels", "inbox")
	if err := m.scanForConversation(inboxRoot, conversationID, &messages); err != nil {
		return nil, fmt.Errorf("failed to scan inbox: %w", err)
	}

	// Scan archive
	archiveRoot := filepath.Join(m.basePath, "channels", "archive")
	if err := m.scanForConversation(archiveRoot, conversationID, &messages); err != nil {
		return nil, fmt.Errorf("failed to scan archive: %w", err)
	}

	// Sort by date
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Date < messages[j].Date
	})

	return messages, nil
}

// Helper methods

// generateMessageID generates a unique message ID.
func (m *messageBusStore) generateMessageID() (string, error) {
	counterFile := filepath.Join(m.basePath, ".hive_counter")

	// Read current counter
	counter := 1
	if data, err := os.ReadFile(counterFile); err == nil {
		if c, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			counter = c + 1
		}
	}

	// Write incremented counter
	if err := os.WriteFile(counterFile, []byte(fmt.Sprintf("%d\n", counter)), 0o644); err != nil {
		return "", fmt.Errorf("failed to write counter file: %w", err)
	}

	return fmt.Sprintf("MSG-%05d", counter), nil
}

// findMessage searches for a message in all inboxes.
func (m *messageBusStore) findMessage(messageID string) (string, error) {
	inboxRoot := filepath.Join(m.basePath, "channels", "inbox")

	var foundPath string
	err := filepath.Walk(inboxRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".yaml") {
			if strings.HasPrefix(info.Name(), messageID) {
				foundPath = path
				return filepath.SkipAll
			}
		}
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", fmt.Errorf("failed to walk inbox directory: %w", err)
	}

	if foundPath == "" {
		return "", fmt.Errorf("message %s not found in any inbox", messageID)
	}

	return foundPath, nil
}

// scanForConversation scans a directory tree for messages matching a conversation ID.
func (m *messageBusStore) scanForConversation(root string, conversationID string, messages *[]models.HiveMessage) error {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".yaml") {
			msg, err := m.readMessage(path)
			if err != nil {
				return fmt.Errorf("failed to read message %s: %w", path, err)
			}
			if msg.ConversationID == conversationID {
				*messages = append(*messages, msg)
			}
		}
		return nil
	})

	return err
}

// readMessage reads a message from a YAML file.
func (m *messageBusStore) readMessage(path string) (models.HiveMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return models.HiveMessage{}, fmt.Errorf("failed to read file: %w", err)
	}

	var msg models.HiveMessage
	if err := yaml.Unmarshal(data, &msg); err != nil {
		return models.HiveMessage{}, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return msg, nil
}

// writeMessage writes a message to a YAML file atomically.
func (m *messageBusStore) writeMessage(path string, msg models.HiveMessage) error {
	data, err := yaml.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write to temporary file first
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath) // Clean up on error
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

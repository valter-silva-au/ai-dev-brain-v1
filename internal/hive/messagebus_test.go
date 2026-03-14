package hive

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
)

func TestMessageBus_PublishAndSubscribe(t *testing.T) {
	basePath := t.TempDir()
	bus := NewMessageBus(basePath)

	msg := models.HiveMessage{
		From:     "team-lead",
		To:       "researcher",
		Subject:  "Research React hooks",
		Content:  "Please research React hooks best practices",
		Type:     models.HiveMessageRequest,
		Priority: "P1",
	}

	// Publish the message
	err := bus.Publish(msg)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Subscribe for recipient
	messages, err := bus.Subscribe("researcher")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Verify message received
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	received := messages[0]

	// Verify auto-generated ID
	if received.ID == "" {
		t.Error("Expected auto-generated ID, got empty string")
	}
	if received.ID != "MSG-00001" {
		t.Errorf("Expected ID MSG-00001, got %s", received.ID)
	}

	// Verify auto-generated date
	if received.Date == "" {
		t.Error("Expected auto-generated date, got empty string")
	}

	// Parse and verify date is recent
	parsedTime, err := time.Parse(time.RFC3339, received.Date)
	if err != nil {
		t.Fatalf("Failed to parse date: %v", err)
	}
	if time.Since(parsedTime) > time.Minute {
		t.Error("Message date is not recent")
	}

	// Verify other fields
	if received.From != msg.From {
		t.Errorf("Expected From=%s, got %s", msg.From, received.From)
	}
	if received.To != msg.To {
		t.Errorf("Expected To=%s, got %s", msg.To, received.To)
	}
	if received.Subject != msg.Subject {
		t.Errorf("Expected Subject=%s, got %s", msg.Subject, received.Subject)
	}
	if received.Content != msg.Content {
		t.Errorf("Expected Content=%s, got %s", msg.Content, received.Content)
	}
}

func TestMessageBus_PublishMultiple(t *testing.T) {
	basePath := t.TempDir()
	bus := NewMessageBus(basePath)

	recipient := "researcher"
	baseTime := time.Now().UTC()

	// Publish 3 messages with different timestamps
	for i := 1; i <= 3; i++ {
		msg := models.HiveMessage{
			From:     "team-lead",
			To:       recipient,
			Subject:  "Task " + string(rune('0'+i)),
			Content:  "Content " + string(rune('0'+i)),
			Type:     models.HiveMessageRequest,
			Priority: "P1",
			Date:     baseTime.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
		}

		err := bus.Publish(msg)
		if err != nil {
			t.Fatalf("Publish %d failed: %v", i, err)
		}
	}

	// Subscribe and verify all 3 returned
	messages, err := bus.Subscribe(recipient)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	// Verify sorted by date (ascending)
	for i := 0; i < len(messages)-1; i++ {
		if messages[i].Date >= messages[i+1].Date {
			t.Errorf("Messages not sorted by date: %s >= %s", messages[i].Date, messages[i+1].Date)
		}
	}

	// Verify subjects are in order
	for i, msg := range messages {
		expectedSubject := "Task " + string(rune('1'+i))
		if msg.Subject != expectedSubject {
			t.Errorf("Message %d: expected Subject=%s, got %s", i, expectedSubject, msg.Subject)
		}
	}
}

func TestMessageBus_SubscribeEmpty(t *testing.T) {
	basePath := t.TempDir()
	bus := NewMessageBus(basePath)

	// Subscribe for nonexistent recipient
	messages, err := bus.Subscribe("nonexistent")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Verify empty slice returned (not nil)
	if messages == nil {
		t.Error("Expected empty slice, got nil")
	}

	if len(messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(messages))
	}
}

func TestMessageBus_SubscribeDifferentRecipients(t *testing.T) {
	basePath := t.TempDir()
	bus := NewMessageBus(basePath)

	// Publish to agent-a
	msgA := models.HiveMessage{
		From:     "team-lead",
		To:       "agent-a",
		Subject:  "Task for A",
		Content:  "Content for A",
		Type:     models.HiveMessageRequest,
		Priority: "P1",
	}
	err := bus.Publish(msgA)
	if err != nil {
		t.Fatalf("Publish to agent-a failed: %v", err)
	}

	// Publish to agent-b
	msgB := models.HiveMessage{
		From:     "team-lead",
		To:       "agent-b",
		Subject:  "Task for B",
		Content:  "Content for B",
		Type:     models.HiveMessageRequest,
		Priority: "P2",
	}
	err = bus.Publish(msgB)
	if err != nil {
		t.Fatalf("Publish to agent-b failed: %v", err)
	}

	// Subscribe for agent-a
	messages, err := bus.Subscribe("agent-a")
	if err != nil {
		t.Fatalf("Subscribe for agent-a failed: %v", err)
	}

	// Verify only agent-a's messages
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message for agent-a, got %d", len(messages))
	}

	if messages[0].To != "agent-a" {
		t.Errorf("Expected To=agent-a, got %s", messages[0].To)
	}
	if messages[0].Subject != "Task for A" {
		t.Errorf("Expected Subject='Task for A', got %s", messages[0].Subject)
	}
}

func TestMessageBus_MarkProcessed(t *testing.T) {
	basePath := t.TempDir()
	bus := NewMessageBus(basePath)

	msg := models.HiveMessage{
		From:     "team-lead",
		To:       "researcher",
		Subject:  "Research React hooks",
		Content:  "Please research React hooks best practices",
		Type:     models.HiveMessageRequest,
		Priority: "P1",
	}

	// Publish message
	err := bus.Publish(msg)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Subscribe to get the message ID
	messages, err := bus.Subscribe("researcher")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	messageID := messages[0].ID

	// Mark processed
	err = bus.MarkProcessed(messageID)
	if err != nil {
		t.Fatalf("MarkProcessed failed: %v", err)
	}

	// Subscribe again
	messages, err = bus.Subscribe("researcher")
	if err != nil {
		t.Fatalf("Subscribe after MarkProcessed failed: %v", err)
	}

	// Verify message no longer in inbox
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages in inbox after MarkProcessed, got %d", len(messages))
	}
}

func TestMessageBus_MarkProcessedCreatesArchive(t *testing.T) {
	basePath := t.TempDir()
	bus := NewMessageBus(basePath)

	msg := models.HiveMessage{
		From:     "team-lead",
		To:       "researcher",
		Subject:  "Research React hooks",
		Content:  "Please research React hooks best practices",
		Type:     models.HiveMessageRequest,
		Priority: "P1",
	}

	// Publish message
	err := bus.Publish(msg)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Subscribe to get the message
	messages, err := bus.Subscribe("researcher")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	messageID := messages[0].ID
	messageDate := messages[0].Date

	// Parse date to get year and month
	parsedTime, err := time.Parse(time.RFC3339, messageDate)
	if err != nil {
		t.Fatalf("Failed to parse date: %v", err)
	}

	year := parsedTime.Format("2006")
	month := parsedTime.Format("01")

	// Mark processed
	err = bus.MarkProcessed(messageID)
	if err != nil {
		t.Fatalf("MarkProcessed failed: %v", err)
	}

	// Verify file exists in archive/{year}/{month}/ directory
	archivePath := filepath.Join(basePath, "channels", "archive", year, month, messageID+".yaml")
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Errorf("Expected archive file at %s, but it does not exist", archivePath)
	}
}

func TestMessageBus_GetConversation(t *testing.T) {
	basePath := t.TempDir()
	bus := NewMessageBus(basePath)

	conversationID := "conv-123"
	baseTime := time.Now().UTC()

	// Publish 3 messages with same ConversationID
	for i := 1; i <= 3; i++ {
		msg := models.HiveMessage{
			ConversationID: conversationID,
			From:           "team-lead",
			To:             "researcher",
			Subject:        "Message " + string(rune('0'+i)),
			Content:        "Content " + string(rune('0'+i)),
			Type:           models.HiveMessageRequest,
			Priority:       "P1",
			Date:           baseTime.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
		}

		err := bus.Publish(msg)
		if err != nil {
			t.Fatalf("Publish %d failed: %v", i, err)
		}
	}

	// Get conversation
	messages, err := bus.GetConversation(conversationID)
	if err != nil {
		t.Fatalf("GetConversation failed: %v", err)
	}

	// Verify all 3 returned
	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages in conversation, got %d", len(messages))
	}

	// Verify all have the same conversation ID
	for i, msg := range messages {
		if msg.ConversationID != conversationID {
			t.Errorf("Message %d: expected ConversationID=%s, got %s", i, conversationID, msg.ConversationID)
		}
	}

	// Verify sorted by date
	for i := 0; i < len(messages)-1; i++ {
		if messages[i].Date >= messages[i+1].Date {
			t.Errorf("Messages not sorted by date: %s >= %s", messages[i].Date, messages[i+1].Date)
		}
	}
}

func TestMessageBus_GetConversationMixed(t *testing.T) {
	basePath := t.TempDir()
	bus := NewMessageBus(basePath)

	conversationID1 := "conv-111"
	conversationID2 := "conv-222"
	baseTime := time.Now().UTC()

	// Publish messages with different ConversationIDs
	msg1 := models.HiveMessage{
		ConversationID: conversationID1,
		From:           "team-lead",
		To:             "researcher",
		Subject:        "Message 1 for conv-111",
		Content:        "Content 1",
		Type:           models.HiveMessageRequest,
		Priority:       "P1",
		Date:           baseTime.Add(1 * time.Minute).Format(time.RFC3339),
	}
	err := bus.Publish(msg1)
	if err != nil {
		t.Fatalf("Publish msg1 failed: %v", err)
	}

	msg2 := models.HiveMessage{
		ConversationID: conversationID2,
		From:           "team-lead",
		To:             "developer",
		Subject:        "Message 1 for conv-222",
		Content:        "Content 2",
		Type:           models.HiveMessageRequest,
		Priority:       "P1",
		Date:           baseTime.Add(2 * time.Minute).Format(time.RFC3339),
	}
	err = bus.Publish(msg2)
	if err != nil {
		t.Fatalf("Publish msg2 failed: %v", err)
	}

	msg3 := models.HiveMessage{
		ConversationID: conversationID1,
		From:           "researcher",
		To:             "team-lead",
		Subject:        "Message 2 for conv-111",
		Content:        "Content 3",
		Type:           models.HiveMessageResponse,
		Priority:       "P1",
		Date:           baseTime.Add(3 * time.Minute).Format(time.RFC3339),
	}
	err = bus.Publish(msg3)
	if err != nil {
		t.Fatalf("Publish msg3 failed: %v", err)
	}

	// Get one conversation
	messages, err := bus.GetConversation(conversationID1)
	if err != nil {
		t.Fatalf("GetConversation failed: %v", err)
	}

	// Verify only matching returned
	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages for conv-111, got %d", len(messages))
	}

	for i, msg := range messages {
		if msg.ConversationID != conversationID1 {
			t.Errorf("Message %d: expected ConversationID=%s, got %s", i, conversationID1, msg.ConversationID)
		}
	}
}

func TestMessageBus_AutoIncrementID(t *testing.T) {
	basePath := t.TempDir()
	bus := NewMessageBus(basePath)

	// Publish 3 messages
	for i := 1; i <= 3; i++ {
		msg := models.HiveMessage{
			From:     "team-lead",
			To:       "researcher",
			Subject:  "Task " + string(rune('0'+i)),
			Content:  "Content " + string(rune('0'+i)),
			Type:     models.HiveMessageRequest,
			Priority: "P1",
		}

		err := bus.Publish(msg)
		if err != nil {
			t.Fatalf("Publish %d failed: %v", i, err)
		}
	}

	// Subscribe and verify IDs
	messages, err := bus.Subscribe("researcher")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	expectedIDs := []string{"MSG-00001", "MSG-00002", "MSG-00003"}
	for i, msg := range messages {
		if msg.ID != expectedIDs[i] {
			t.Errorf("Message %d: expected ID=%s, got %s", i, expectedIDs[i], msg.ID)
		}
	}
}

func TestMessageBus_CounterPersistence(t *testing.T) {
	basePath := t.TempDir()

	// Create first MessageBus and publish a message
	bus1 := NewMessageBus(basePath)

	msg1 := models.HiveMessage{
		From:     "team-lead",
		To:       "researcher",
		Subject:  "First message",
		Content:  "Content 1",
		Type:     models.HiveMessageRequest,
		Priority: "P1",
	}

	err := bus1.Publish(msg1)
	if err != nil {
		t.Fatalf("Publish with bus1 failed: %v", err)
	}

	// Create new MessageBus at same path
	bus2 := NewMessageBus(basePath)

	// Publish another message
	msg2 := models.HiveMessage{
		From:     "team-lead",
		To:       "developer",
		Subject:  "Second message",
		Content:  "Content 2",
		Type:     models.HiveMessageRequest,
		Priority: "P1",
	}

	err = bus2.Publish(msg2)
	if err != nil {
		t.Fatalf("Publish with bus2 failed: %v", err)
	}

	// Subscribe and verify ID continues from previous counter
	messages1, err := bus2.Subscribe("researcher")
	if err != nil {
		t.Fatalf("Subscribe for researcher failed: %v", err)
	}

	messages2, err := bus2.Subscribe("developer")
	if err != nil {
		t.Fatalf("Subscribe for developer failed: %v", err)
	}

	if len(messages1) != 1 {
		t.Fatalf("Expected 1 message for researcher, got %d", len(messages1))
	}
	if len(messages2) != 1 {
		t.Fatalf("Expected 1 message for developer, got %d", len(messages2))
	}

	// Verify IDs
	if messages1[0].ID != "MSG-00001" {
		t.Errorf("Expected first message ID=MSG-00001, got %s", messages1[0].ID)
	}
	if messages2[0].ID != "MSG-00002" {
		t.Errorf("Expected second message ID=MSG-00002, got %s", messages2[0].ID)
	}
}

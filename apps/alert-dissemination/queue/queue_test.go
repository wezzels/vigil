package queue

import (
	"testing"
	"time"
)

// TestNewAlertQueue tests queue creation
func TestNewAlertQueue(t *testing.T) {
	queue := NewAlertQueue()
	if queue == nil {
		t.Fatal("NewAlertQueue() returned nil")
	}
	if queue.Size() != 0 {
		t.Errorf("Expected empty queue, got size %d", queue.Size())
	}
}

// TestEnqueueDequeue tests basic enqueue/dequeue
func TestEnqueueDequeue(t *testing.T) {
	queue := NewAlertQueue()

	alert1 := &Alert{
		ID:       "ALERT-001",
		Priority: PriorityHigh,
		Content:  "Test alert",
	}

	err := queue.Enqueue(alert1)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	if queue.Size() != 1 {
		t.Errorf("Expected size 1, got %d", queue.Size())
	}

	alert2, err := queue.Dequeue()
	if err != nil {
		t.Fatalf("Dequeue() error = %v", err)
	}

	if alert2.ID != alert1.ID {
		t.Errorf("Expected ID %s, got %s", alert1.ID, alert2.ID)
	}

	if queue.Size() != 0 {
		t.Errorf("Expected empty queue, got size %d", queue.Size())
	}
}

// TestPriorityOrdering tests priority ordering
func TestPriorityOrdering(t *testing.T) {
	queue := NewAlertQueue()

	alerts := []*Alert{
		{ID: "LOW", Priority: PriorityLow, Content: "Low priority"},
		{ID: "HIGH", Priority: PriorityHigh, Content: "High priority"},
		{ID: "NORMAL", Priority: PriorityNormal, Content: "Normal priority"},
		{ID: "CRITICAL", Priority: PriorityCritical, Content: "Critical priority"},
	}

	for _, a := range alerts {
		queue.Enqueue(a)
	}

	// Should dequeue in order: CRITICAL, HIGH, NORMAL, LOW
	expectedOrder := []string{"CRITICAL", "HIGH", "NORMAL", "LOW"}
	for _, expected := range expectedOrder {
		alert, err := queue.Dequeue()
		if err != nil {
			t.Fatalf("Dequeue() error = %v", err)
		}
		if alert.ID != expected {
			t.Errorf("Expected ID %s, got %s", expected, alert.ID)
		}
	}
}

// TestFIFOOrdering tests FIFO within same priority
func TestFIFOOrdering(t *testing.T) {
	queue := NewAlertQueue()

	now := time.Now()
	alerts := []*Alert{
		{ID: "FIRST", Priority: PriorityHigh, Content: "First high", CreatedAt: now},
		{ID: "SECOND", Priority: PriorityHigh, Content: "Second high", CreatedAt: now.Add(1 * time.Millisecond)},
		{ID: "THIRD", Priority: PriorityHigh, Content: "Third high", CreatedAt: now.Add(2 * time.Millisecond)},
	}

	for _, a := range alerts {
		queue.Enqueue(a)
	}

	// Should dequeue in FIFO order for same priority
	for _, expected := range []string{"FIRST", "SECOND", "THIRD"} {
		alert, err := queue.Dequeue()
		if err != nil {
			t.Fatalf("Dequeue() error = %v", err)
		}
		if alert.ID != expected {
			t.Errorf("Expected ID %s, got %s", expected, alert.ID)
		}
	}
}

// TestMixedPriorityFIFO tests FIFO with mixed priorities
func TestMixedPriorityFIFO(t *testing.T) {
	queue := NewAlertQueue()

	now := time.Now()
	alerts := []*Alert{
		{ID: "HIGH-1", Priority: PriorityHigh, Content: "High 1", CreatedAt: now},
		{ID: "NORMAL-1", Priority: PriorityNormal, Content: "Normal 1", CreatedAt: now.Add(1 * time.Millisecond)},
		{ID: "HIGH-2", Priority: PriorityHigh, Content: "High 2", CreatedAt: now.Add(2 * time.Millisecond)},
		{ID: "NORMAL-2", Priority: PriorityNormal, Content: "Normal 2", CreatedAt: now.Add(3 * time.Millisecond)},
	}

	for _, a := range alerts {
		queue.Enqueue(a)
	}

	// HIGH-1, HIGH-2 (FIFO within priority), then NORMAL-1, NORMAL-2
	expectedOrder := []string{"HIGH-1", "HIGH-2", "NORMAL-1", "NORMAL-2"}
	for _, expected := range expectedOrder {
		alert, err := queue.Dequeue()
		if err != nil {
			t.Fatalf("Dequeue() error = %v", err)
		}
		if alert.ID != expected {
			t.Errorf("Expected ID %s, got %s", expected, alert.ID)
		}
	}
}

// TestDequeueEmpty tests dequeue on empty queue
func TestDequeueEmpty(t *testing.T) {
	queue := NewAlertQueue()

	_, err := queue.Dequeue()
	if err != ErrQueueEmpty {
		t.Errorf("Expected ErrQueueEmpty, got %v", err)
	}
}

// TestPeek tests peek without removal
func TestPeek(t *testing.T) {
	queue := NewAlertQueue()

	alert1 := &Alert{ID: "ALERT-001", Priority: PriorityHigh, Content: "Test"}
	queue.Enqueue(alert1)

	alert2, err := queue.Peek()
	if err != nil {
		t.Fatalf("Peek() error = %v", err)
	}
	if alert2.ID != alert1.ID {
		t.Errorf("Expected ID %s, got %s", alert1.ID, alert2.ID)
	}

	// Queue should still have the item
	if queue.Size() != 1 {
		t.Errorf("Expected size 1 after peek, got %d", queue.Size())
	}
}

// TestPeekEmpty tests peek on empty queue
func TestPeekEmpty(t *testing.T) {
	queue := NewAlertQueue()

	_, err := queue.Peek()
	if err != ErrQueueEmpty {
		t.Errorf("Expected ErrQueueEmpty, got %v", err)
	}
}

// TestRemove tests removal by ID
func TestRemove(t *testing.T) {
	queue := NewAlertQueue()

	alerts := []*Alert{
		{ID: "ALERT-001", Priority: PriorityHigh, Content: "Test 1"},
		{ID: "ALERT-002", Priority: PriorityHigh, Content: "Test 2"},
		{ID: "ALERT-003", Priority: PriorityHigh, Content: "Test 3"},
	}

	for _, a := range alerts {
		queue.Enqueue(a)
	}

	removed, err := queue.Remove("ALERT-002")
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if removed.ID != "ALERT-002" {
		t.Errorf("Expected ID ALERT-002, got %s", removed.ID)
	}

	if queue.Size() != 2 {
		t.Errorf("Expected size 2 after remove, got %d", queue.Size())
	}

	// Remove non-existent
	_, err = queue.Remove("NOT-EXIST")
	if err != ErrAlertNotFound {
		t.Errorf("Expected ErrAlertNotFound, got %v", err)
	}
}

// TestGet tests retrieval by ID
func TestGet(t *testing.T) {
	queue := NewAlertQueue()

	alerts := []*Alert{
		{ID: "ALERT-001", Priority: PriorityHigh, Content: "Test 1"},
		{ID: "ALERT-002", Priority: PriorityHigh, Content: "Test 2"},
	}

	for _, a := range alerts {
		queue.Enqueue(a)
	}

	alert, err := queue.Get("ALERT-001")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if alert.ID != "ALERT-001" {
		t.Errorf("Expected ID ALERT-001, got %s", alert.ID)
	}

	// Queue should still have all items
	if queue.Size() != 2 {
		t.Errorf("Expected size 2 after get, got %d", queue.Size())
	}
}

// TestList tests listing all alerts
func TestList(t *testing.T) {
	queue := NewAlertQueue()

	alerts := []*Alert{
		{ID: "ALERT-001", Priority: PriorityHigh, Content: "Test 1"},
		{ID: "ALERT-002", Priority: PriorityNormal, Content: "Test 2"},
	}

	for _, a := range alerts {
		queue.Enqueue(a)
	}

	list := queue.List()
	if len(list) != 2 {
		t.Errorf("Expected 2 items in list, got %d", len(list))
	}
}

// TestListByPriority tests listing by priority
func TestListByPriority(t *testing.T) {
	queue := NewAlertQueue()

	alerts := []*Alert{
		{ID: "HIGH-1", Priority: PriorityHigh, Content: "High 1"},
		{ID: "NORMAL-1", Priority: PriorityNormal, Content: "Normal 1"},
		{ID: "HIGH-2", Priority: PriorityHigh, Content: "High 2"},
	}

	for _, a := range alerts {
		queue.Enqueue(a)
	}

	highList := queue.ListByPriority(PriorityHigh)
	if len(highList) != 2 {
		t.Errorf("Expected 2 high priority items, got %d", len(highList))
	}

	normalList := queue.ListByPriority(PriorityNormal)
	if len(normalList) != 1 {
		t.Errorf("Expected 1 normal priority item, got %d", len(normalList))
	}
}

// TestPurgeExpired tests purging expired alerts
func TestPurgeExpired(t *testing.T) {
	queue := NewAlertQueue()

	now := time.Now()
	alerts := []*Alert{
		{ID: "VALID", Priority: PriorityHigh, Content: "Valid", ExpiresAt: now.Add(1 * time.Hour)},
		{ID: "EXPIRED-1", Priority: PriorityHigh, Content: "Expired 1", ExpiresAt: now.Add(-1 * time.Hour)},
		{ID: "EXPIRED-2", Priority: PriorityNormal, Content: "Expired 2", ExpiresAt: now.Add(-1 * time.Minute)},
	}

	for _, a := range alerts {
		queue.Enqueue(a)
	}

	purged := queue.PurgeExpired()
	if purged != 2 {
		t.Errorf("Expected 2 purged, got %d", purged)
	}

	if queue.Size() != 1 {
		t.Errorf("Expected size 1 after purge, got %d", queue.Size())
	}
}

// TestClear tests clearing the queue
func TestClear(t *testing.T) {
	queue := NewAlertQueue()

	for i := 0; i < 10; i++ {
		queue.Enqueue(&Alert{
			ID:       string(rune('A' + i)),
			Priority: PriorityNormal,
			Content:  "Test",
		})
	}

	queue.Clear()
	if queue.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", queue.Size())
	}
}

// TestStats tests queue statistics
func TestStats(t *testing.T) {
	queue := NewAlertQueue()

	alerts := []*Alert{
		{ID: "HIGH-1", Priority: PriorityHigh, Content: "High 1"},
		{ID: "HIGH-2", Priority: PriorityHigh, Content: "High 2"},
		{ID: "NORMAL-1", Priority: PriorityNormal, Content: "Normal 1"},
		{ID: "CRITICAL-1", Priority: PriorityCritical, Content: "Critical"},
	}

	for _, a := range alerts {
		queue.Enqueue(a)
	}

	stats := queue.Stats()
	if stats.Total != 4 {
		t.Errorf("Expected total 4, got %d", stats.Total)
	}

	if stats.ByPriority[PriorityHigh] != 2 {
		t.Errorf("Expected 2 high, got %d", stats.ByPriority[PriorityHigh])
	}

	if stats.ByPriority[PriorityCritical] != 1 {
		t.Errorf("Expected 1 critical, got %d", stats.ByPriority[PriorityCritical])
	}
}

// TestBatchQueue tests batch queue operations
func TestBatchQueue(t *testing.T) {
	bq := NewBatchQueue()

	alerts := []*Alert{
		{ID: "ALERT-001", Priority: PriorityHigh, Content: "Test 1", Recipient: "RECIPIENT-A"},
		{ID: "ALERT-002", Priority: PriorityNormal, Content: "Test 2", Recipient: "RECIPIENT-A"},
		{ID: "ALERT-003", Priority: PriorityCritical, Content: "Test 3", Recipient: "RECIPIENT-B"},
		{ID: "ALERT-004", Priority: PriorityLow, Content: "Default", Recipient: ""},
	}

	for _, a := range alerts {
		err := bq.Enqueue(a)
		if err != nil {
			t.Fatalf("Enqueue() error = %v", err)
		}
	}

	// Check RECIPIENT-A queue
	alert, err := bq.Dequeue("RECIPIENT-A")
	if err != nil {
		t.Fatalf("Dequeue() error = %v", err)
	}
	if alert.ID != "ALERT-001" { // High priority first
		t.Errorf("Expected ALERT-001, got %s", alert.ID)
	}

	// Check RECIPIENT-B queue
	alert, err = bq.Dequeue("RECIPIENT-B")
	if err != nil {
		t.Fatalf("Dequeue() error = %v", err)
	}
	if alert.ID != "ALERT-003" {
		t.Errorf("Expected ALERT-003, got %s", alert.ID)
	}

	// Check total size
	if bq.TotalSize() != 2 { // ALERT-002 and ALERT-004
		t.Errorf("Expected total size 2, got %d", bq.TotalSize())
	}
}

// TestBatchQueuePurgeExpired tests batch queue purging
func TestBatchQueuePurgeExpired(t *testing.T) {
	bq := NewBatchQueue()

	now := time.Now()
	alerts := []*Alert{
		{ID: "VALID", Priority: PriorityHigh, Content: "Valid", Recipient: "A", ExpiresAt: now.Add(1 * time.Hour)},
		{ID: "EXPIRED", Priority: PriorityNormal, Content: "Expired", Recipient: "A", ExpiresAt: now.Add(-1 * time.Hour)},
	}

	for _, a := range alerts {
		bq.Enqueue(a)
	}

	purged := bq.PurgeExpired()
	if purged != 1 {
		t.Errorf("Expected 1 purged, got %d", purged)
	}

	if bq.TotalSize() != 1 {
		t.Errorf("Expected total size 1 after purge, got %d", bq.TotalSize())
	}
}

// TestPriorityStrings tests priority string conversion
func TestPriorityStrings(t *testing.T) {
	tests := []struct {
		priority AlertPriority
		expected string
	}{
		{PriorityLow, "LOW"},
		{PriorityNormal, "NORMAL"},
		{PriorityHigh, "HIGH"},
		{PriorityCritical, "CRITICAL"},
		{PriorityImminent, "IMMINENT"},
	}

	for _, tt := range tests {
		result := GetPriorityString(tt.priority)
		if result != tt.expected {
			t.Errorf("GetPriorityString(%d) = %s, want %s", tt.priority, result, tt.expected)
		}
	}
}

// TestParsePriority tests priority parsing
func TestParsePriority(t *testing.T) {
	tests := []struct {
		input    string
		expected AlertPriority
	}{
		{"LOW", PriorityLow},
		{"low", PriorityLow},
		{"NORMAL", PriorityNormal},
		{"normal", PriorityNormal},
		{"HIGH", PriorityHigh},
		{"high", PriorityHigh},
		{"CRITICAL", PriorityCritical},
		{"critical", PriorityCritical},
		{"IMMINENT", PriorityImminent},
		{"imminent", PriorityImminent},
	}

	for _, tt := range tests {
		result, err := ParsePriority(tt.input)
		if err != nil {
			t.Errorf("ParsePriority(%s) error = %v", tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("ParsePriority(%s) = %d, want %d", tt.input, result, tt.expected)
		}
	}

	// Test invalid
	_, err := ParsePriority("INVALID")
	if err == nil {
		t.Error("Expected error for invalid priority")
	}
}

// BenchmarkEnqueue benchmarks enqueue operations
func BenchmarkEnqueue(b *testing.B) {
	queue := NewAlertQueue()
	alert := &Alert{
		ID:       "ALERT-001",
		Priority: PriorityHigh,
		Content:  "Test alert",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alert.ID = string(rune(i))
		queue.Enqueue(alert)
	}
}

// BenchmarkDequeue benchmarks dequeue operations
func BenchmarkDequeue(b *testing.B) {
	queue := NewAlertQueue()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.Enqueue(&Alert{
			ID:       string(rune(i)),
			Priority: AlertPriority(i % 5),
			Content:  "Test",
		})
		_, _ = queue.Dequeue()
	}
}

// BenchmarkMixedOperations benchmarks mixed operations
func BenchmarkMixedOperations(b *testing.B) {
	queue := NewAlertQueue()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			queue.Enqueue(&Alert{
				ID:       string(rune(i)),
				Priority: PriorityHigh,
				Content:  "Test",
			})
		} else {
			_, _ = queue.Dequeue()
		}
	}
}

// Package queue provides a priority queue for alert dissemination
// with FIFO ordering within priority levels
package queue

import (
	"container/heap"
	"sync"
	"time"
)

// AlertPriority defines alert priority levels
type AlertPriority int

const (
	PriorityLow      AlertPriority = 0
	PriorityNormal    AlertPriority = 1
	PriorityHigh      AlertPriority = 2
	PriorityCritical  AlertPriority = 3
	PriorityImminent  AlertPriority = 4
)

// Alert represents an alert in the queue
type Alert struct {
	ID          string        `json:"id"`
	Priority    AlertPriority `json:"priority"`
	Category    string        `json:"category"`
	Content     string        `json:"content"`
	Source      string        `json:"source"`
	CreatedAt   time.Time     `json:"created_at"`
	ExpiresAt   time.Time     `json:"expires_at,omitempty"`
	Attempts    int           `json:"attempts"`
	MaxAttempts int           `json:"max_attempts"`
	Recipient   string        `json:"recipient,omitempty"`
	index       int           // Index in heap (managed by heap)
}

// AlertQueue is a thread-safe priority queue for alerts
type AlertQueue struct {
	items []*Alert
	mutex sync.RWMutex
}

// NewAlertQueue creates a new alert queue
func NewAlertQueue() *AlertQueue {
	aq := &AlertQueue{
		items: make([]*Alert, 0),
	}
	heap.Init(aq)
	return aq
}

// Len implements heap.Interface
func (aq *AlertQueue) Len() int {
	return len(aq.items)
}

// Less implements heap.Interface
// Higher priority comes first, then FIFO within same priority
func (aq *AlertQueue) Less(i, j int) bool {
	if aq.items[i].Priority != aq.items[j].Priority {
		return aq.items[i].Priority > aq.items[j].Priority
	}
	// FIFO within same priority - earlier created_at first
	return aq.items[i].CreatedAt.Before(aq.items[j].CreatedAt)
}

// Swap implements heap.Interface
func (aq *AlertQueue) Swap(i, j int) {
	aq.items[i], aq.items[j] = aq.items[j], aq.items[i]
	aq.items[i].index = i
	aq.items[j].index = j
}

// Push implements heap.Interface
func (aq *AlertQueue) Push(x interface{}) {
	n := len(aq.items)
	item := x.(*Alert)
	item.index = n
	aq.items = append(aq.items, item)
}

// Pop implements heap.Interface
func (aq *AlertQueue) Pop() interface{} {
	n := len(aq.items)
	item := aq.items[n-1]
	item.index = -1 // Mark as removed
	aq.items = aq.items[0 : n-1]
	return item
}

// Enqueue adds an alert to the queue (thread-safe)
func (aq *AlertQueue) Enqueue(alert *Alert) error {
	aq.mutex.Lock()
	defer aq.mutex.Unlock()

	// Set defaults
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now()
	}
	if alert.MaxAttempts == 0 {
		alert.MaxAttempts = 3
	}

	heap.Push(aq, alert)
	return nil
}

// Dequeue removes and returns the highest priority alert (thread-safe)
func (aq *AlertQueue) Dequeue() (*Alert, error) {
	aq.mutex.Lock()
	defer aq.mutex.Unlock()

	if aq.Len() == 0 {
		return nil, ErrQueueEmpty
	}

	item := heap.Pop(aq)
	return item.(*Alert), nil
}

// Peek returns the highest priority alert without removing it
func (aq *AlertQueue) Peek() (*Alert, error) {
	aq.mutex.RLock()
	defer aq.mutex.RUnlock()

	if aq.Len() == 0 {
		return nil, ErrQueueEmpty
	}

	return aq.items[0], nil
}

// Size returns the queue size
func (aq *AlertQueue) Size() int {
	aq.mutex.RLock()
	defer aq.mutex.RUnlock()
	return aq.Len()
}

// Clear removes all items from the queue
func (aq *AlertQueue) Clear() {
	aq.mutex.Lock()
	defer aq.mutex.Unlock()
	aq.items = make([]*Alert, 0)
}

// Remove removes an alert by ID
func (aq *AlertQueue) Remove(id string) (*Alert, error) {
	aq.mutex.Lock()
	defer aq.mutex.Unlock()

	for i, item := range aq.items {
		if item.ID == id {
			return heap.Remove(aq, i).(*Alert), nil
		}
	}

	return nil, ErrAlertNotFound
}

// Get returns an alert by ID without removing it
func (aq *AlertQueue) Get(id string) (*Alert, error) {
	aq.mutex.RLock()
	defer aq.mutex.RUnlock()

	for _, item := range aq.items {
		if item.ID == id {
			return item, nil
		}
	}

	return nil, ErrAlertNotFound
}

// List returns all alerts in the queue (ordered by priority)
func (aq *AlertQueue) List() []*Alert {
	aq.mutex.RLock()
	defer aq.mutex.RUnlock()

	result := make([]*Alert, len(aq.items))
	copy(result, aq.items)
	return result
}

// ListByPriority returns alerts filtered by priority
func (aq *AlertQueue) ListByPriority(priority AlertPriority) []*Alert {
	aq.mutex.RLock()
	defer aq.mutex.RUnlock()

	var result []*Alert
	for _, item := range aq.items {
		if item.Priority == priority {
			result = append(result, item)
		}
	}
	return result
}

// PurgeExpired removes all expired alerts from the queue
func (aq *AlertQueue) PurgeExpired() int {
	aq.mutex.Lock()
	defer aq.mutex.Unlock()

	now := time.Now()
	count := 0
	newItems := make([]*Alert, 0, len(aq.items))

	for _, item := range aq.items {
		if !item.ExpiresAt.IsZero() && item.ExpiresAt.Before(now) {
			count++
		} else {
			newItems = append(newItems, item)
		}
	}

	aq.items = newItems
	heap.Init(aq)
	return count
}

// Stats returns queue statistics
func (aq *AlertQueue) Stats() QueueStats {
	aq.mutex.RLock()
	defer aq.mutex.RUnlock()

	stats := QueueStats{
		Total: aq.Len(),
		ByPriority: make(map[AlertPriority]int),
	}

	for _, item := range aq.items {
		stats.ByPriority[item.Priority]++
	}

	return stats
}

// QueueStats holds queue statistics
type QueueStats struct {
	Total      int                    `json:"total"`
	ByPriority map[AlertPriority]int  `json:"by_priority"`
}

// Errors
var (
	ErrQueueEmpty     = &QueueError{Code: "QUEUE_EMPTY", Message: "queue is empty"}
	ErrAlertNotFound  = &QueueError{Code: "ALERT_NOT_FOUND", Message: "alert not found"}
	ErrInvalidPriority = &QueueError{Code: "INVALID_PRIORITY", Message: "invalid priority level"}
)

// QueueError represents a queue error
type QueueError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *QueueError) Error() string {
	return e.Message
}

// GetPriorityString returns string representation of priority
func GetPriorityString(p AlertPriority) string {
	switch p {
	case PriorityLow:
		return "LOW"
	case PriorityNormal:
		return "NORMAL"
	case PriorityHigh:
		return "HIGH"
	case PriorityCritical:
		return "CRITICAL"
	case PriorityImminent:
		return "IMMINENT"
	default:
		return "UNKNOWN"
	}
}

// ParsePriority parses a priority string
func ParsePriority(s string) (AlertPriority, error) {
	switch s {
	case "LOW", "low":
		return PriorityLow, nil
	case "NORMAL", "normal":
		return PriorityNormal, nil
	case "HIGH", "high":
		return PriorityHigh, nil
	case "CRITICAL", "critical":
		return PriorityCritical, nil
	case "IMMINENT", "imminent":
		return PriorityImminent, nil
	default:
		return PriorityLow, ErrInvalidPriority
	}
}

// BatchQueue handles multiple alert queues by recipient
type BatchQueue struct {
	queues map[string]*AlertQueue
	mutex  sync.RWMutex
}

// NewBatchQueue creates a new batch queue
func NewBatchQueue() *BatchQueue {
	return &BatchQueue{
		queues: make(map[string]*AlertQueue),
	}
}

// Enqueue adds an alert to the recipient's queue
func (bq *BatchQueue) Enqueue(alert *Alert) error {
	bq.mutex.Lock()
	defer bq.mutex.Unlock()

	recipient := alert.Recipient
	if recipient == "" {
		recipient = "default"
	}

	if _, exists := bq.queues[recipient]; !exists {
		bq.queues[recipient] = NewAlertQueue()
	}

	return bq.queues[recipient].Enqueue(alert)
}

// Dequeue removes an alert from a recipient's queue
func (bq *BatchQueue) Dequeue(recipient string) (*Alert, error) {
	bq.mutex.RLock()
	defer bq.mutex.RUnlock()

	queue, exists := bq.queues[recipient]
	if !exists {
		return nil, ErrQueueEmpty
	}

	return queue.Dequeue()
}

// GetQueue returns a recipient's queue
func (bq *BatchQueue) GetQueue(recipient string) (*AlertQueue, bool) {
	bq.mutex.RLock()
	defer bq.mutex.RUnlock()
	queue, exists := bq.queues[recipient]
	return queue, exists
}

// AllQueues returns all recipient queues
func (bq *BatchQueue) AllQueues() map[string]*AlertQueue {
	bq.mutex.RLock()
	defer bq.mutex.RUnlock()
	result := make(map[string]*AlertQueue)
	for k, v := range bq.queues {
		result[k] = v
	}
	return result
}

// TotalSize returns total alerts across all queues
func (bq *BatchQueue) TotalSize() int {
	bq.mutex.RLock()
	defer bq.mutex.RUnlock()
	total := 0
	for _, queue := range bq.queues {
		total += queue.Size()
	}
	return total
}

// PurgeExpired purges expired alerts from all queues
func (bq *BatchQueue) PurgeExpired() int {
	bq.mutex.Lock()
	defer bq.mutex.Unlock()
	total := 0
	for _, queue := range bq.queues {
		total += queue.PurgeExpired()
	}
	return total
}
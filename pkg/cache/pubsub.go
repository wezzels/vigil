// Package cache provides pub/sub coordination
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// MessageType defines the type of pub/sub message
type MessageType string

const (
	MessageTypeTrack      MessageType = "track"
	MessageTypeAlert      MessageType = "alert"
	MessageTypeEvent      MessageType = "event"
	MessageTypeCommand    MessageType = "command"
	MessageTypeHeartbeat  MessageType = "heartbeat"
	MessageTypeLeader     MessageType = "leader"
)

// Message represents a pub/sub message
type Message struct {
	Type      MessageType     `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// PubSub provides pub/sub coordination
type PubSub struct {
	cache   *Cache
	subscriptions map[string][]chan *Message
	mu      sync.RWMutex
}

// NewPubSub creates a new pub/sub coordinator
func NewPubSub(cache *Cache) *PubSub {
	return &PubSub{
		cache:        cache,
		subscriptions: make(map[string][]chan *Message),
	}
}

// Publish publishes a message to a channel
func (ps *PubSub) Publish(ctx context.Context, channel string, msgType MessageType, source string, payload interface{}) error {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := &Message{
		Type:      msgType,
		Source:    source,
		Timestamp: time.Now(),
		Payload:   payloadData,
	}

	return ps.cache.Publish(ctx, channel, msg)
}

// Subscribe subscribes to a channel
func (ps *PubSub) Subscribe(ctx context.Context, channel string) (<-chan *Message, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ch := make(chan *Message, 100)
	ps.subscriptions[channel] = append(ps.subscriptions[channel], ch)

	// Start Redis subscription in background
	go func() {
		sub := ps.cache.Subscribe(ctx, channel)
		defer close(ch)
		defer sub.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-sub.Channel():
				if !ok {
					return
				}

				var message Message
				if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
					continue
				}

				// Send to local subscribers
				ps.mu.RLock()
				subs := ps.subscriptions[channel]
				ps.mu.RUnlock()

				for _, sub := range subs {
					select {
					case sub <- &message:
					default:
						// Channel full, skip
					}
				}
			}
		}
	}()

	return ch, nil
}

// Unsubscribe unsubscribes from a channel
func (ps *PubSub) Unsubscribe(ctx context.Context, channel string, ch <-chan *Message) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	subs := ps.subscriptions[channel]
	newSubs := make([]chan *Message, 0)
	for _, sub := range subs {
		if sub != ch {
			newSubs = append(newSubs, sub)
		} else {
			close(sub)
		}
	}
	ps.subscriptions[channel] = newSubs
}

// Broadcast sends a message to all subscribers
func (ps *PubSub) Broadcast(ctx context.Context, msgType MessageType, source string, payload interface{}) error {
	return ps.Publish(ctx, "broadcast", msgType, source, payload)
}

// LeaderElection provides leader election using Redis
type LeaderElection struct {
	cache    *Cache
	key      string
	id       string
	ttl      time.Duration
	leader   bool
	stopChan chan struct{}
	mu       sync.RWMutex
}

// NewLeaderElection creates a new leader election
func NewLeaderElection(cache *Cache, key, id string, ttl time.Duration) *LeaderElection {
	return &LeaderElection{
		cache:    cache,
		key:      key,
		id:       id,
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}
}

// Campaign campaigns for leadership
func (le *LeaderElection) Campaign(ctx context.Context) error {
	// Try to set key with NX (only if not exists)
	key := le.leaderKey()
	
	// Use SET NX EX for atomic operation
	err := le.cache.SetString(ctx, key, le.id, le.ttl)
	if err != nil {
		return err
	}

	le.mu.Lock()
	le.leader = true
	le.mu.Unlock()

	// Start keep-alive
	go le.keepAlive()

	return nil
}

// Resign resigns from leadership
func (le *LeaderElection) Resign(ctx context.Context) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	if !le.leader {
		return nil
	}

	// Stop keep-alive
	close(le.stopChan)

	// Delete key
	le.mu.RLock()
	err := le.cache.Delete(ctx, le.leaderKey())
	le.leader = false
	le.mu.RUnlock()

	return err
}

// IsLeader returns true if this instance is the leader
func (le *LeaderElection) IsLeader() bool {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.leader
}

// GetLeader returns the current leader ID
func (le *LeaderElection) GetLeader(ctx context.Context) (string, error) {
	key := le.leaderKey()
	return le.cache.Get(ctx, key)
}

// keepAlive maintains leadership
func (le *LeaderElection) keepAlive() {
	ticker := time.NewTicker(le.ttl / 2)
	defer ticker.Stop()

	for {
		select {
		case <-le.stopChan:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			key := le.leaderKey()
			
			// Check if we still hold the lock
			value, err := le.cache.Get(ctx, key)
			if err != nil || value != le.id {
				le.mu.Lock()
				le.leader = false
				le.mu.Unlock()
				cancel()
				return
			}

			// Extend TTL
			le.cache.Expire(ctx, key, le.ttl)
			cancel()
		}
	}
}

// leaderKey returns the leader election key
func (le *LeaderElection) leaderKey() string {
	return fmt.Sprintf("leader:%s", le.key)
}

// Coordinator coordinates multiple instances
type Coordinator struct {
	pubsub   *PubSub
	election *LeaderElection
	id       string
	channels []string
}

// NewCoordinator creates a new coordinator
func NewCoordinator(cache *Cache, id string, electionTTL time.Duration) *Coordinator {
	return &Coordinator{
		pubsub:   NewPubSub(cache),
		election: NewLeaderElection(cache, "coordinator", id, electionTTL),
		id:       id,
		channels: []string{},
	}
}

// Join joins the coordination group
func (c *Coordinator) Join(ctx context.Context, channels ...string) error {
	c.channels = channels

	// Subscribe to channels
	for _, channel := range channels {
		_, err := c.pubsub.Subscribe(ctx, channel)
		if err != nil {
			return err
		}
	}

	// Campaign for leadership
	return c.election.Campaign(ctx)
}

// Leave leaves the coordination group
func (c *Coordinator) Leave(ctx context.Context) error {
	return c.election.Resign(ctx)
}

// IsLeader returns true if this instance is the leader
func (c *Coordinator) IsLeader() bool {
	return c.election.IsLeader()
}

// Publish publishes a message
func (c *Coordinator) Publish(ctx context.Context, channel string, msgType MessageType, payload interface{}) error {
	return c.pubsub.Publish(ctx, channel, msgType, c.id, payload)
}

// Broadcast broadcasts to all channels
func (c *Coordinator) Broadcast(ctx context.Context, msgType MessageType, payload interface{}) error {
	return c.pubsub.Broadcast(ctx, msgType, c.id, payload)
}

// GetLeader returns the current leader
func (c *Coordinator) GetLeader(ctx context.Context) (string, error) {
	return c.election.GetLeader(ctx)
}
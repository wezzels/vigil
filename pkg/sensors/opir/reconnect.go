// Package opir provides reconnection logic with circuit breaker
package opir

import (
	"context"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// StateClosed means the circuit is closed (normal operation)
	StateClosed CircuitState = iota
	// StateOpen means the circuit is open (failing)
	StateOpen
	// StateHalfOpen means the circuit is half-open (testing)
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	maxFailures      int
	timeout          time.Duration
	successThreshold int

	failures    int
	lastFailure time.Time
	state       CircuitState
	mu          sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:      maxFailures,
		timeout:          timeout,
		successThreshold: 3,
		state:            StateClosed,
	}
}

// Allow checks if a request should be allowed
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.failures = 0
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.failures = 0
	case StateHalfOpen:
		cb.failures++
		if cb.failures >= cb.successThreshold {
			cb.state = StateClosed
			cb.failures = 0
		}
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.maxFailures {
			cb.state = StateOpen
		}
	case StateHalfOpen:
		cb.state = StateOpen
	}
}

// State returns the current circuit state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reconnector handles connection reconnection with backoff
type Reconnector struct {
	config  *OPIRConfig
	circuit *CircuitBreaker

	attempts    int
	lastAttempt time.Time
	mu          sync.RWMutex
}

// NewReconnector creates a new reconnector
func NewReconnector(config *OPIRConfig) *Reconnector {
	return &Reconnector{
		config:  config,
		circuit: NewCircuitBreaker(5, 30*time.Second),
	}
}

// ShouldReconnect determines if reconnection should be attempted
func (r *Reconnector) ShouldReconnect() bool {
	if !r.circuit.Allow() {
		return false
	}

	r.mu.RLock()
	attempts := r.attempts
	r.mu.RUnlock()

	return attempts < r.config.MaxRetries
}

// NextBackoff calculates the next backoff duration
func (r *Reconnector) NextBackoff() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.attempts++

	// Exponential backoff with jitter
	baseDelay := r.config.RetryDelay
	maxDelay := r.config.MaxRetryDelay

	delay := baseDelay * time.Duration(1<<uint(r.attempts-1))
	if delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter (±10%)
	jitter := time.Duration(float64(delay) * 0.1)
	delay = delay + jitter - time.Duration(float64(jitter)*2*float64(time.Now().UnixNano()%1000)/1000)

	r.lastAttempt = time.Now()

	return delay
}

// RecordSuccess records a successful connection
func (r *Reconnector) RecordSuccess() {
	r.mu.Lock()
	r.attempts = 0
	r.mu.Unlock()
	r.circuit.RecordSuccess()
}

// RecordFailure records a failed connection
func (r *Reconnector) RecordFailure() {
	r.circuit.RecordFailure()
}

// Reset resets the reconnector
func (r *Reconnector) Reset() {
	r.mu.Lock()
	r.attempts = 0
	r.lastAttempt = time.Time{}
	r.mu.Unlock()
}

// Attempts returns the number of connection attempts
func (r *Reconnector) Attempts() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.attempts
}

// CircuitState returns the circuit breaker state
func (r *Reconnector) CircuitState() CircuitState {
	return r.circuit.State()
}

// HealthChecker provides health checking functionality
type HealthChecker struct {
	config    *OPIRConfig
	lastCheck time.Time
	healthy   bool
	mu        sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(config *OPIRConfig) *HealthChecker {
	return &HealthChecker{
		config:  config,
		healthy: true,
	}
}

// Check performs a health check
func (h *HealthChecker) Check(ctx context.Context, feed OPIRDataFeed) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.lastCheck = time.Now()

	// Check if feed is connected
	if !feed.IsConnected() {
		h.healthy = false
		return NewConnectionError("feed not connected", true)
	}

	// Check feed stats
	stats := feed.Stats()

	// Check error rate
	if stats.TotalReceived > 0 {
		errorRate := float64(stats.TotalErrors) / float64(stats.TotalReceived)
		if errorRate > 0.1 { // More than 10% errors
			h.healthy = false
			return NewConnectionError("high error rate", true)
		}
	}

	// Check receive rate
	if stats.ReceiveRate < 1.0 { // Less than 1 message/second
		h.healthy = false
		return NewConnectionError("low receive rate", true)
	}

	h.healthy = true
	return nil
}

// IsHealthy returns the health status
func (h *HealthChecker) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.healthy
}

// LastCheck returns the time of the last health check
func (h *HealthChecker) LastCheck() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastCheck
}

// ConnectionPool manages multiple connections
type ConnectionPool struct {
	feeds   []OPIRDataFeed
	current int
	mu      sync.RWMutex
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(feeds ...OPIRDataFeed) *ConnectionPool {
	return &ConnectionPool{
		feeds:   feeds,
		current: 0,
	}
}

// Get returns the next available feed
func (p *ConnectionPool) Get() OPIRDataFeed {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.feeds) == 0 {
		return nil
	}

	feed := p.feeds[p.current]
	p.current = (p.current + 1) % len(p.feeds)

	return feed
}

// GetHealthy returns a healthy feed
func (p *ConnectionPool) GetHealthy() OPIRDataFeed {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for i := 0; i < len(p.feeds); i++ {
		idx := (p.current + i) % len(p.feeds)
		if p.feeds[idx].IsConnected() {
			return p.feeds[idx]
		}
	}

	return nil
}

// All returns all feeds
func (p *ConnectionPool) All() []OPIRDataFeed {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.feeds
}

// Add adds a feed to the pool
func (p *ConnectionPool) Add(feed OPIRDataFeed) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.feeds = append(p.feeds, feed)
}

// Remove removes a feed from the pool
func (p *ConnectionPool) Remove(feed OPIRDataFeed) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, f := range p.feeds {
		if f == feed {
			p.feeds = append(p.feeds[:i], p.feeds[i+1:]...)
			break
		}
	}
}

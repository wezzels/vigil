// Package ha provides high availability components for VIGIL
package ha

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HealthStatus represents health check status
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusUnknown   HealthStatus = "unknown"
)

// HealthCheck represents a health check
type HealthCheck struct {
	Name      string        `json:"name"`
	Status    HealthStatus  `json:"status"`
	Message   string        `json:"message,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
}

// HealthChecker defines the health check interface
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) (*HealthCheck, error)
}

// HealthProbe provides health probe functionality
type HealthProbe struct {
	checkers  map[string]HealthChecker
	mu        sync.RWMutex
	startTime time.Time
}

// NewHealthProbe creates a new health probe
func NewHealthProbe() *HealthProbe {
	return &HealthProbe{
		checkers:  make(map[string]HealthChecker),
		startTime: time.Now(),
	}
}

// Register registers a health checker
func (hp *HealthProbe) Register(checker HealthChecker) {
	hp.mu.Lock()
	defer hp.mu.Unlock()
	hp.checkers[checker.Name()] = checker
}

// Liveness checks if the service is alive
func (hp *HealthProbe) Liveness(ctx context.Context) *HealthCheck {
	return &HealthCheck{
		Name:      "liveness",
		Status:    StatusHealthy,
		Timestamp: time.Now(),
	}
}

// Readiness checks if the service is ready to accept traffic
func (hp *HealthProbe) Readiness(ctx context.Context) *HealthCheck {
	hp.mu.RLock()
	defer hp.mu.RUnlock()

	checks := make([]*HealthCheck, 0, len(hp.checkers))
	allHealthy := true

	for _, checker := range hp.checkers {
		check, err := checker.Check(ctx)
		if err != nil {
			check = &HealthCheck{
				Name:      checker.Name(),
				Status:    StatusUnhealthy,
				Message:   err.Error(),
				Timestamp: time.Now(),
			}
			allHealthy = false
		}
		if check != nil && check.Status != StatusHealthy {
			allHealthy = false
		}
		checks = append(checks, check)
	}

	status := StatusHealthy
	if !allHealthy {
		status = StatusUnhealthy
	}

	return &HealthCheck{
		Name:      "readiness",
		Status:    status,
		Timestamp: time.Now(),
	}
}

// Startup checks if the service has started
func (hp *HealthProbe) Startup(ctx context.Context, minUptime time.Duration) *HealthCheck {
	uptime := time.Since(hp.startTime)
	
	if uptime < minUptime {
		return &HealthCheck{
			Name:      "startup",
			Status:    StatusUnhealthy,
			Message:   fmt.Sprintf("service not ready (uptime: %v, required: %v)", uptime, minUptime),
			Timestamp: time.Now(),
		}
	}

	return &HealthCheck{
		Name:      "startup",
		Status:    StatusHealthy,
		Timestamp: time.Now(),
	}
}

// CheckAll runs all health checks
func (hp *HealthProbe) CheckAll(ctx context.Context) map[string]*HealthCheck {
	hp.mu.RLock()
	defer hp.mu.RUnlock()

	results := make(map[string]*HealthCheck)
	for name, checker := range hp.checkers {
		check, err := checker.Check(ctx)
		if err != nil {
			check = &HealthCheck{
				Name:      name,
				Status:    StatusUnhealthy,
				Message:   err.Error(),
				Timestamp: time.Now(),
			}
		}
		results[name] = check
	}

	return results
}

// ShutdownState represents shutdown state
type ShutdownState int

const (
	StateRunning ShutdownState = iota
	StateShuttingDown
	StateTerminated
)

// GracefulShutdown provides graceful shutdown functionality
type GracefulShutdown struct {
	state       ShutdownState
	connections sync.WaitGroup
	mu          sync.RWMutex
	stopChan    chan struct{}
	timeout     time.Duration
	handlers    []func(context.Context) error
}

// NewGracefulShutdown creates a new graceful shutdown handler
func NewGracefulShutdown(timeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		stopChan: make(chan struct{}),
		timeout: timeout,
	}
}

// AddConnection adds a connection to track
func (gs *GracefulShutdown) AddConnection() {
	gs.connections.Add(1)
}

// RemoveConnection removes a connection from tracking
func (gs *GracefulShutdown) RemoveConnection() {
	gs.connections.Done()
}

// IsRunning returns true if the service is running
func (gs *GracefulShutdown) IsRunning() bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.state == StateRunning
}

// IsShuttingDown returns true if the service is shutting down
func (gs *GracefulShutdown) IsShuttingDown() bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.state == StateShuttingDown
}

// RegisterHandler registers a shutdown handler
func (gs *GracefulShutdown) RegisterHandler(handler func(context.Context) error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.handlers = append(gs.handlers, handler)
}

// Shutdown initiates graceful shutdown
func (gs *GracefulShutdown) Shutdown(ctx context.Context) error {
	gs.mu.Lock()
	gs.state = StateShuttingDown
	close(gs.stopChan)
	gs.mu.Unlock()

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, gs.timeout)
	defer cancel()

	// Run shutdown handlers
	for _, handler := range gs.handlers {
		if err := handler(ctx); err != nil {
			// Log error but continue
		}
	}

	// Wait for connections to drain
	done := make(chan struct{})
	go func() {
		gs.connections.Wait()
		close(done)
	}()

	select {
	case <-done:
		gs.mu.Lock()
		gs.state = StateTerminated
		gs.mu.Unlock()
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout")
	}
}

// StopChan returns the stop channel
func (gs *GracefulShutdown) StopChan() <-chan struct{} {
	return gs.stopChan
}

// WaitForShutdown waits for shutdown signal
func (gs *GracefulShutdown) WaitForShutdown(sigs ...interface{}) {
	// This would typically be used with signal.Notify
	// Implementation depends on signal handling
}

// CircuitBreaker provides circuit breaker pattern
type CircuitBreaker struct {
	name          string
	failureCount  int
	successCount  int
	failureLimit  int
	successLimit  int
	timeout       time.Duration
	lastFailure   time.Time
	state         CircuitState
	mu            sync.RWMutex
}

// CircuitState represents circuit breaker state
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, failureLimit, successLimit int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:         name,
		failureLimit: failureLimit,
		successLimit: successLimit,
		timeout:      timeout,
		state:        StateClosed,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if !cb.Allow() {
		return fmt.Errorf("circuit breaker %s is open", cb.name)
	}

	err := fn()
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

// Allow checks if requests are allowed
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailure) > cb.timeout {
			// Transition to half-open
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case StateHalfOpen:
		return true
	}
	return false
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	cb.successCount++

	if cb.state == StateHalfOpen && cb.successCount >= cb.successLimit {
		cb.state = StateClosed
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailure = time.Now()

	if cb.failureCount >= cb.failureLimit {
		cb.state = StateOpen
	}
}

// State returns the current state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}
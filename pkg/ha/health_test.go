package ha

import (
	"context"
	"testing"
	"time"
)

// MockHealthChecker is a mock health checker for testing
type MockHealthChecker struct {
	name    string
	healthy bool
	err     error
}

func (m *MockHealthChecker) Name() string {
	return m.name
}

func (m *MockHealthChecker) Check(ctx context.Context) (*HealthCheck, error) {
	if m.err != nil {
		return nil, m.err
	}
	status := StatusHealthy
	if !m.healthy {
		status = StatusUnhealthy
	}
	return &HealthCheck{
		Name:      m.name,
		Status:    status,
		Timestamp: time.Now(),
	}, nil
}

// TestNewHealthProbe tests health probe creation
func TestNewHealthProbe(t *testing.T) {
	hp := NewHealthProbe()
	if hp == nil {
		t.Fatal("Expected health probe, got nil")
	}

	if hp.checkers == nil {
		t.Error("Expected checkers map to be initialized")
	}
}

// TestHealthProbeRegister tests health checker registration
func TestHealthProbeRegister(t *testing.T) {
	hp := NewHealthProbe()
	checker := &MockHealthChecker{name: "test", healthy: true}

	hp.Register(checker)

	if len(hp.checkers) != 1 {
		t.Errorf("Expected 1 checker, got %d", len(hp.checkers))
	}

	if hp.checkers["test"] != checker {
		t.Error("Expected checker to be registered")
	}
}

// TestHealthProbeLiveness tests liveness probe
func TestHealthProbeLiveness(t *testing.T) {
	hp := NewHealthProbe()
	ctx := context.Background()

	check := hp.Liveness(ctx)

	if check.Name != "liveness" {
		t.Errorf("Expected name liveness, got %s", check.Name)
	}

	if check.Status != StatusHealthy {
		t.Errorf("Expected status healthy, got %s", check.Status)
	}
}

// TestHealthProbeReadiness tests readiness probe
func TestHealthProbeReadiness(t *testing.T) {
	hp := NewHealthProbe()
	ctx := context.Background()

	// Register healthy checker
	healthyChecker := &MockHealthChecker{name: "healthy", healthy: true}
	hp.Register(healthyChecker)

	check := hp.Readiness(ctx)

	if check.Name != "readiness" {
		t.Errorf("Expected name readiness, got %s", check.Name)
	}

	if check.Status != StatusHealthy {
		t.Errorf("Expected status healthy, got %s", check.Status)
	}
}

// TestHealthProbeReadinessUnhealthy tests readiness with unhealthy checker
func TestHealthProbeReadinessUnhealthy(t *testing.T) {
	hp := NewHealthProbe()
	ctx := context.Background()

	// Register unhealthy checker
	unhealthyChecker := &MockHealthChecker{name: "unhealthy", healthy: false}
	hp.Register(unhealthyChecker)

	check := hp.Readiness(ctx)

	if check.Status != StatusUnhealthy {
		t.Errorf("Expected status unhealthy, got %s", check.Status)
	}
}

// TestHealthProbeStartup tests startup probe
func TestHealthProbeStartup(t *testing.T) {
	hp := NewHealthProbe()
	ctx := context.Background()

	// Test with min uptime not yet reached
	check := hp.Startup(ctx, 10*time.Hour)

	if check.Status != StatusUnhealthy {
		t.Errorf("Expected status unhealthy, got %s", check.Status)
	}

	// Test with min uptime reached (0 means always ready)
	hp.startTime = time.Now().Add(-1 * time.Hour)
	check = hp.Startup(ctx, 0)

	if check.Status != StatusHealthy {
		t.Errorf("Expected status healthy, got %s", check.Status)
	}
}

// TestHealthProbeCheckAll tests checking all health checkers
func TestHealthProbeCheckAll(t *testing.T) {
	hp := NewHealthProbe()
	ctx := context.Background()

	// Register multiple checkers
	hp.Register(&MockHealthChecker{name: "checker1", healthy: true})
	hp.Register(&MockHealthChecker{name: "checker2", healthy: true})
	hp.Register(&MockHealthChecker{name: "checker3", healthy: false})

	results := hp.CheckAll(ctx)

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Check that results exist for all checkers
	for _, name := range []string{"checker1", "checker2", "checker3"} {
		if _, ok := results[name]; !ok {
			t.Errorf("Expected result for %s", name)
		}
	}
}

// TestNewGracefulShutdown tests graceful shutdown creation
func TestNewGracefulShutdown(t *testing.T) {
	gs := NewGracefulShutdown(30 * time.Second)

	if gs == nil {
		t.Fatal("Expected graceful shutdown, got nil")
	}

	if gs.timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", gs.timeout)
	}

	if gs.state != StateRunning {
		t.Errorf("Expected initial state running, got %d", gs.state)
	}
}

// TestGracefulShutdownIsRunning tests running state check
func TestGracefulShutdownIsRunning(t *testing.T) {
	gs := NewGracefulShutdown(30 * time.Second)

	if !gs.IsRunning() {
		t.Error("Expected IsRunning() to return true")
	}
}

// TestGracefulShutdownConnections tests connection tracking
func TestGracefulShutdownConnections(t *testing.T) {
	gs := NewGracefulShutdown(30 * time.Second)

	// Add connections
	gs.AddConnection()
	gs.AddConnection()
	gs.AddConnection()

	// Remove connections
	gs.RemoveConnection()
	gs.RemoveConnection()
	gs.RemoveConnection()

	// Connections should be zero
	// This is implicit - WaitGroup would panic if we remove too many
}

// TestGracefulShutdownStopChan tests stop channel
func TestGracefulShutdownStopChan(t *testing.T) {
	gs := NewGracefulShutdown(30 * time.Second)

	stopChan := gs.StopChan()
	if stopChan == nil {
		t.Error("Expected stop channel, got nil")
	}
}

// TestNewCircuitBreaker tests circuit breaker creation
func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker("test", 3, 2, 5*time.Second)

	if cb == nil {
		t.Fatal("Expected circuit breaker, got nil")
	}

	if cb.name != "test" {
		t.Errorf("Expected name test, got %s", cb.name)
	}

	if cb.failureLimit != 3 {
		t.Errorf("Expected failure limit 3, got %d", cb.failureLimit)
	}

	if cb.successLimit != 2 {
		t.Errorf("Expected success limit 2, got %d", cb.successLimit)
	}
}

// TestCircuitBreakerAllow tests circuit breaker allow
func TestCircuitBreakerAllow(t *testing.T) {
	cb := NewCircuitBreaker("test", 3, 2, 5*time.Second)

	// Initially should allow
	if !cb.Allow() {
		t.Error("Expected Allow() to return true when circuit is closed")
	}
}

// TestCircuitBreakerRecordSuccess tests recording success
func TestCircuitBreakerRecordSuccess(t *testing.T) {
	cb := NewCircuitBreaker("test", 3, 2, 5*time.Second)

	// Record failures to open circuit
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != StateOpen {
		t.Error("Expected circuit to be open after failures")
	}

	// Record successes to close circuit (need to simulate half-open state)
	cb.mu.Lock()
	cb.state = StateHalfOpen
	cb.successCount = 0
	cb.mu.Unlock()

	cb.RecordSuccess()
	cb.RecordSuccess()

	// After successLimit successes, circuit should close
	if cb.State() != StateClosed {
		t.Errorf("Expected circuit to be closed after successes, got %d", cb.State())
	}
}

// TestCircuitBreakerRecordFailure tests recording failure
func TestCircuitBreakerRecordFailure(t *testing.T) {
	cb := NewCircuitBreaker("test", 3, 2, 5*time.Second)

	// Record failures
	cb.RecordFailure()
	if cb.State() != StateClosed {
		t.Error("Circuit should still be closed after 1 failure")
	}

	cb.RecordFailure()
	if cb.State() != StateClosed {
		t.Error("Circuit should still be closed after 2 failures")
	}

	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Error("Circuit should be open after 3 failures")
	}
}

// TestCircuitBreakerState tests state transitions
func TestCircuitBreakerState(t *testing.T) {
	cb := NewCircuitBreaker("test", 3, 2, 5*time.Second)

	// Initial state
	if cb.State() != StateClosed {
		t.Errorf("Expected initial state closed, got %d", cb.State())
	}

	// Record failures to open
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != StateOpen {
		t.Errorf("Expected state open, got %d", cb.State())
	}

	// Simulate half-open
	cb.mu.Lock()
	cb.state = StateHalfOpen
	cb.mu.Unlock()

	if cb.State() != StateHalfOpen {
		t.Errorf("Expected state half-open, got %d", cb.State())
	}
}

// TestHealthCheckStruct tests health check structure
func TestHealthCheckStruct(t *testing.T) {
	check := &HealthCheck{
		Name:      "test",
		Status:    StatusHealthy,
		Message:   "All good",
		Timestamp: time.Now(),
		Duration:  100 * time.Millisecond,
	}

	if check.Name != "test" {
		t.Errorf("Expected name test, got %s", check.Name)
	}

	if check.Status != StatusHealthy {
		t.Errorf("Expected status healthy, got %s", check.Status)
	}
}

// TestCircuitBreakerExecute tests execute function
func TestCircuitBreakerExecute(t *testing.T) {
	cb := NewCircuitBreaker("test", 3, 2, 5*time.Second)
	ctx := context.Background()

	// Successful execution
	err := cb.Execute(ctx, func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Failed execution
	err = cb.Execute(ctx, func() error {
		return context.DeadlineExceeded
	})
	if err == nil {
		t.Error("Expected error from failed execution")
	}
}

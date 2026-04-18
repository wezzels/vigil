package interceptor

import (
	"testing"
	"time"
)

func TestEngagementCoordinator(t *testing.T) {
	ec := NewEngagementCoordinator(nil)

	// Register a shooter
	shooter := &Shooter{
		ID:           "shooter-1",
		Name:         "Battery 1",
		Location:     [3]float64{0, 0, 0},
		Interceptors: []InterceptorType{THAAD, PATRIOT_PAC3},
		Status:       SHOOTER_AVAILABLE,
		BatteryState: BatteryState{
			RoundsReady: 4,
			RoundsTotal: 4,
			RadarActive: true,
		},
	}
	ec.RegisterShooter(shooter)

	// Create engagement order
	order := &EngagementOrder{
		TrackID:         "track-123",
		ShooterID:       "shooter-1",
		InterceptorType: THAAD,
		Strategy:        STRATEGY_LOOK_DOWN_SHLK,
		Priority:        PRIORITY_HIGH,
		LaunchPoint:     [3]float64{0, 0, 0},
		Timeline:        30 * time.Second,
	}
	id, err := ec.CreateEngagementOrder(order)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}
	if id == "" {
		t.Fatal("Order ID should not be empty")
	}

	// Verify order created
	gotOrder := ec.GetOrder(id)
	if gotOrder == nil {
		t.Fatal("Order not found")
	}
	if gotOrder.State != ENG_STATE_PENDING {
		t.Errorf("Expected PENDING state, got %v", gotOrder.State)
	}

	// Authorize
	err = ec.AuthorizeEngagement(id)
	if err != nil {
		t.Fatalf("Failed to authorize: %v", err)
	}

	// Launch
	err = ec.LaunchInterceptor(id)
	if err != nil {
		t.Fatalf("Failed to launch: %v", err)
	}

	gotOrder = ec.GetOrder(id)
	if gotOrder.State != ENG_STATE_LAUNCHED {
		t.Errorf("Expected LAUNCHED state, got %v", gotOrder.State)
	}
	if gotOrder.Attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", gotOrder.Attempts)
	}

	// Check shooter status updated
	s := ec.GetShooter("shooter-1")
	if s.BatteryState.RoundsReady != 3 {
		t.Errorf("Expected 3 rounds ready, got %d", s.BatteryState.RoundsReady)
	}
}

func TestEngagementStateTransitions(t *testing.T) {
	ec := NewEngagementCoordinator(nil)

	order := &EngagementOrder{
		TrackID:         "track-1",
		ShooterID:       "shooter-1",
		InterceptorType: GMD_GBI,
	}
	id, _ := ec.CreateEngagementOrder(order)

	tests := []struct {
		name      string
		fromState EngagementState
		toState   EngagementState
		action    func(string) error
		shouldErr bool
	}{
		{
			name:      "PENDING -> AUTHORIZED",
			fromState: ENG_STATE_PENDING,
			toState:   ENG_STATE_AUTHORIZED,
			action:    ec.AuthorizeEngagement,
			shouldErr: false,
		},
		{
			name:      "AUTHORIZED -> LAUNCHED",
			fromState: ENG_STATE_AUTHORIZED,
			toState:   ENG_STATE_LAUNCHED,
			action:    ec.LaunchInterceptor,
			shouldErr: false,
		},
		{
			name:      "LAUNCHED -> INTERCEPT (success)",
			fromState: ENG_STATE_LAUNCHED,
			toState:   ENG_STATE_INTERCEPT,
			action:    func(id string) error { return ec.ReportIntercept(id, true) },
			shouldErr: false,
		},
		{
			name:      "INTERCEPT -> SUCCESS",
			fromState: ENG_STATE_INTERCEPT,
			toState:   ENG_STATE_SUCCESS,
			action:    func(id string) error { return ec.AssessKill(id, KILL_LEVEL_CATK, 0.95) },
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset order state
			o := ec.GetOrder(id)
			o.State = tt.fromState

			err := tt.action(id)
			if tt.shouldErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			got := ec.GetOrder(id)
			if got.State != tt.toState {
				t.Errorf("Expected state %v, got %v", tt.toState, got.State)
			}
		})
	}
}

func TestAbortEngagement(t *testing.T) {
	ec := NewEngagementCoordinator(nil)

	// Register shooter first
	ec.RegisterShooter(&Shooter{
		ID:           "shooter-1",
		Interceptors: []InterceptorType{THAAD},
		Status:       SHOOTER_AVAILABLE,
		BatteryState: BatteryState{RoundsReady: 1},
	})

	order := &EngagementOrder{
		TrackID:         "track-1",
		ShooterID:       "shooter-1",
		InterceptorType: THAAD,
	}
	id, _ := ec.CreateEngagementOrder(order)

	// Launch then abort
	ec.LaunchInterceptor(id)
	err := ec.AbortEngagement(id, "Threat friendly")
	if err != nil {
		t.Fatalf("Failed to abort: %v", err)
	}

	o := ec.GetOrder(id)
	if o.State != ENG_STATE_ABORTED {
		t.Errorf("Expected ABORTED state, got %v", o.State)
	}
	if o.AbortReason != "Threat friendly" {
		t.Errorf("Expected abort reason 'Threat friendly', got %s", o.AbortReason)
	}
}

func TestCleanupOldOrders(t *testing.T) {
	ec := NewEngagementCoordinator(nil)

	// Create a success order
	order := &EngagementOrder{TrackID: "t1", ShooterID: "s1"}
	id1, _ := ec.CreateEngagementOrder(order)
	o1 := ec.GetOrder(id1)
	o1.State = ENG_STATE_SUCCESS
	o1.UpdatedAt = time.Now().Add(-1 * time.Hour)

	// Create a failed order
	order2 := &EngagementOrder{TrackID: "t2", ShooterID: "s1"}
	id2, _ := ec.CreateEngagementOrder(order2)
	o2 := ec.GetOrder(id2)
	o2.State = ENG_STATE_FAILED
	o2.UpdatedAt = time.Now().Add(-1 * time.Hour)

	// Create an active order
	order3 := &EngagementOrder{TrackID: "t3", ShooterID: "s1"}
	_, _ = ec.CreateEngagementOrder(order3)

	count := ec.CleanupOldOrders(30 * time.Minute)
	if count != 2 {
		t.Errorf("Expected cleanup of 2 orders, got %d", count)
	}

	// Active order should remain
	if ec.GetOrder(id1) != nil || ec.GetOrder(id2) != nil {
		t.Error("Terminal orders should have been cleaned")
	}
}

func TestSimulateEngagement(t *testing.T) {
	ec := NewEngagementCoordinator(nil)

	threatPos := [3]float64{100e3, 0, 0} // 100km away
	threatVel := [3]float64{-3000, 0, 0} // Heading toward origin
	shooterPos := [3]float64{0, 0, 0}

	intercept, level, conf := ec.SimulateEngagement(
		threatPos, threatVel, THREAT_SRBM, shooterPos, THAAD,
	)

	// THAAD vs SRBM at 100km should result in intercept attempt
	t.Logf("Intercept: %v, Level: %v, Confidence: %f", intercept, level, conf)

	// Just verify it runs without error
	if intercept && (level == KILL_LEVEL_NONE || conf == 0) {
		t.Error("Intercept should have a valid kill assessment")
	}
}

func TestEngagementStats(t *testing.T) {
	ec := NewEngagementCoordinator(nil)

	// Create various orders
	states := []EngagementState{
		ENG_STATE_PENDING,
		ENG_STATE_AUTHORIZED,
		ENG_STATE_LAUNCHED,
		ENG_STATE_SUCCESS,
		ENG_STATE_FAILED,
		ENG_STATE_ABORTED,
	}

	for i, state := range states {
		order := &EngagementOrder{TrackID: "t1"}
		id, _ := ec.CreateEngagementOrder(order)
		o := ec.GetOrder(id)
		o.State = state
		_ = i // suppress unused warning
	}

	stats := ec.EngagementStats()
	if stats.TotalOrders != 6 {
		t.Errorf("Expected 6 total orders, got %d", stats.TotalOrders)
	}
	if stats.ActiveEngagements != 3 {
		t.Errorf("Expected 3 active (PENDING+AUTHORIZED+LAUNCHED), got %d", stats.ActiveEngagements)
	}
	if stats.SuccessfulIntercepts != 1 {
		t.Errorf("Expected 1 success, got %d", stats.SuccessfulIntercepts)
	}
}

func TestSelectShooter(t *testing.T) {
	ec := NewEngagementCoordinator(nil)

	// Register GMD battery with GBI (5000km range)
	ec.RegisterShooter(&Shooter{
		ID:           "gmd-battery",
		Location:     [3]float64{0, 0, 0},
		Interceptors: []InterceptorType{GMD_GBI, SM3_IIA},
		Status:       SHOOTER_AVAILABLE,
		BatteryState: BatteryState{RoundsReady: 2},
	})

	// GBI has 5500km range, so 2000km is well within range
	threatPos := [3]float64{2000e3, 0, 0}
	threatVel := [3]float64{-7000, 0, 0}

	shooter, intType, err := ec.SelectShooter(
		threatPos, threatVel,
		THREAT_ICBM,
		500e3, 2000e3, 180, // 180s = midcourse window for GBI
		[]InterceptorType{GMD_GBI},
	)

	if err != nil {
		t.Fatalf("SelectShooter failed: %v", err)
	}

	if shooter == nil {
		t.Fatal("No shooter selected")
	}

	if shooter.ID != "gmd-battery" {
		t.Errorf("Expected gmd-battery, got %s", shooter.ID)
	}

	t.Logf("Selected %s with interceptor type %v", shooter.ID, intType)
}

func TestGetAvailableShooters(t *testing.T) {
	ec := NewEngagementCoordinator(nil)

	ec.RegisterShooter(&Shooter{
		ID:     "available",
		Status: SHOOTER_AVAILABLE,
		BatteryState: BatteryState{RoundsReady: 1},
	})
	ec.RegisterShooter(&Shooter{
		ID:     "committed",
		Status: SHOOTER_COMMITTED,
		BatteryState: BatteryState{RoundsReady: 1},
	})
	ec.RegisterShooter(&Shooter{
		ID:     "norounds",
		Status: SHOOTER_AVAILABLE,
		BatteryState: BatteryState{RoundsReady: 0},
	})

	available := ec.GetAvailableShooters(0, 0)
	if len(available) != 1 {
		t.Errorf("Expected 1 available shooter, got %d", len(available))
	}
	if available[0].ID != "available" {
		t.Errorf("Expected 'available' shooter, got %s", available[0].ID)
	}
}

func TestEngagementErrors(t *testing.T) {
	ec := NewEngagementCoordinator(nil)

	// Test order not found
	err := ec.AuthorizeEngagement("nonexistent")
	if err != ErrOrderNotFound {
		t.Errorf("Expected ErrOrderNotFound, got %v", err)
	}

	// Test invalid state transition
	order := &EngagementOrder{TrackID: "t1", ShooterID: "s1"}
	id, _ := ec.CreateEngagementOrder(order)
	ec.AuthorizeEngagement(id)
	ec.LaunchInterceptor(id)

	// Can't launch again
	err = ec.LaunchInterceptor(id)
	if err != ErrInvalidStateTransition {
		t.Errorf("Expected ErrInvalidStateTransition, got %v", err)
	}
}
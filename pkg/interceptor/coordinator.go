// Package interceptor provides BMD interceptor guidance, kinematics, kill assessment,
// and engagement coordination for GBI, SM-3, THAAD, and Patriot interceptors
package interceptor

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// EngagementState represents the state of an engagement
type EngagementState int

const (
	ENG_STATE_PENDING EngagementState = iota // Awaiting NCA/command authority approval
	ENG_STATE_AUTHORIZED                     // NCA authorized, ready to launch
	ENG_STATE_LAUNCHED                       // Interceptor launched
	ENG_STATE_INTERCEPT                      // Intercept attempted
	ENG_STATE_ASSESS                         // Assessing kill
	ENG_STATE_SUCCESS                        // Successful intercept
	ENG_STATE_FAILED                         // Intercept failed
	ENG_STATE_ABORTED                        // Engagement aborted
)

// ShootStrategy defines engagement geometry
type ShootStrategy int

const (
	STRATEGY_LOOK_DOWN_SHLK ShootStrategy = iota // Shoot-look-shoot (fire, assess, fire again if needed)
	STRATEGY_LOOK_DOWN_ONE                       // Single shot, terminal phase
	STRATEGY_BORE_SHOOT                         // Direct intercept, no tracking
	STRATEGY_SPLASH                             // Area denial
)

// EngagementOrder represents an order to engage a threat
type EngagementOrder struct {
	ID              string            `json:"engagement_id"`
	TrackID         string            `json:"track_id"`
	ShooterID       string            `json:"shooter_id"`
	InterceptorType InterceptorType  `json:"interceptor_type"`
	Strategy        ShootStrategy     `json:"strategy"`
	Priority        PriorityLevel     `json:"priority"`
	LaunchPoint     [3]float64        `json:"launch_point"` // ECEF, meters
	LaunchTime      time.Time         `json:"launch_time"`
	Timeline        time.Duration     `json:"timeline"` // Time from now to intercept window
	State           EngagementState   `json:"state"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Attempts        int               `json:"attempts"`
	MaxAttempts     int               `json:"max_attempts"`
	AbortReason     string            `json:"abort_reason,omitempty"`
}

// PriorityLevel for engagement ordering
type PriorityLevel int

const (
	PRIORITY_LOW PriorityLevel = iota
	PRIORITY_MEDIUM
	PRIORITY_HIGH
	PRIORITY_CRITICAL
)

// Shooter represents a firing platform / battery
type Shooter struct {
	ID            string            `json:"shooter_id"`
	Name          string            `json:"name"`
	Location      [3]float64        `json:"location"` // ECEF, meters
	Interceptors  []InterceptorType  `json:"interceptors"` // Available interceptor types
	Status        ShooterStatus      `json:"status"`
	BatteryState  BatteryState       `json:"battery_state"`
}

// ShooterStatus availability of shooter
type ShooterStatus int

const (
	SHOOTER_AVAILABLE ShooterStatus = iota
	SHOOTER_COMMITTED
	SHOOTER_FIRING
	SHOOTER_RELOADING
	SHOOTER_MAINTENANCE
	SHOOTER_KILLED
)

// BatteryState of a shooter battery
type BatteryState struct {
	RoundsReady    int     `json:"rounds_ready"`
	RoundsTotal    int     `json:"rounds_total"`
	RadarActive    bool    `json:"radar_active"`
	TrackFilePos   [3]float64 `json:"track_file_pos"`
	EngagementActive bool   `json:"engagement_active"`
}

// EngagementCoordinator manages engagement lifecycle
type EngagementCoordinator struct {
	config    *CoordinatorConfig
	orders    map[string]*EngagementOrder
	shooters  map[string]*Shooter
	shootersMu sync.RWMutex
	ordersMu  sync.RWMutex
	nextOrderID uint64
}

// CoordinatorConfig for engagement management
type CoordinatorConfig struct {
	MaxConcurrentEngagements int           `json:"max_concurrent_engagements"`
	DefaultTimeline          time.Duration `json:"default_timeline"`
	MaxAttempts              int           `json:"max_attempts"`
	AssessmentTimeout        time.Duration `json:"assessment_timeout"`
	AbortCooldown            time.Duration `json:"abort_cooldown"`
	RequireNATApproval       bool          `json:"require_nca_approval"`
}

// DefaultCoordinatorConfig returns standard configuration
func DefaultCoordinatorConfig() *CoordinatorConfig {
	return &CoordinatorConfig{
		MaxConcurrentEngagements: 10,
		DefaultTimeline:          60 * time.Second,
		MaxAttempts:              3,
		AssessmentTimeout:        30 * time.Second,
		AbortCooldown:            10 * time.Second,
		RequireNATApproval:      true,
	}
}

// NewEngagementCoordinator creates a coordinator
func NewEngagementCoordinator(config *CoordinatorConfig) *EngagementCoordinator {
	if config == nil {
		config = DefaultCoordinatorConfig()
	}
	return &EngagementCoordinator{
		config:   config,
		orders:   make(map[string]*EngagementOrder),
		shooters: make(map[string]*Shooter),
	}
}

// RegisterShooter adds a firing platform
func (ec *EngagementCoordinator) RegisterShooter(shooter *Shooter) {
	ec.shootersMu.Lock()
	defer ec.shootersMu.Unlock()
	ec.shooters[shooter.ID] = shooter
}

// GetShooter returns shooter by ID
func (ec *EngagementCoordinator) GetShooter(id string) *Shooter {
	ec.shootersMu.RLock()
	defer ec.shootersMu.RUnlock()
	return ec.shooters[id]
}

// GetAvailableShooters returns shooters that can engage
func (ec *EngagementCoordinator) GetAvailableShooters(threatAlt, threatRange float64) []*Shooter {
	ec.shootersMu.RLock()
	defer ec.shootersMu.RUnlock()

	available := make([]*Shooter, 0)
	for _, s := range ec.shooters {
		if s.Status != SHOOTER_AVAILABLE {
			continue
		}
		if s.BatteryState.RoundsReady <= 0 {
			continue
		}
		available = append(available, s)
	}
	return available
}

// CreateEngagementOrder creates a new engagement order
func (ec *EngagementCoordinator) CreateEngagementOrder(order *EngagementOrder) (string, error) {
	ec.ordersMu.Lock()
	defer ec.ordersMu.Unlock()

	// Check capacity
	if len(ec.orders) >= ec.config.MaxConcurrentEngagements {
		return "", ErrEngagementFull
	}

	// Generate ID
	ec.nextOrderID++
	order.ID = fmt.Sprintf("%016x", ec.nextOrderID)

	// Initialize state
	order.State = ENG_STATE_PENDING
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	order.Attempts = 0
	order.MaxAttempts = ec.config.MaxAttempts

	ec.orders[order.ID] = order
	return order.ID, nil
}

// GetOrder returns order by ID
func (ec *EngagementCoordinator) GetOrder(id string) *EngagementOrder {
	ec.ordersMu.RLock()
	defer ec.ordersMu.RUnlock()
	return ec.orders[id]
}

// GetAllOrders returns all engagement orders
func (ec *EngagementCoordinator) GetAllOrders() []*EngagementOrder {
	ec.ordersMu.RLock()
	defer ec.ordersMu.RUnlock()

	orders := make([]*EngagementOrder, 0, len(ec.orders))
	for _, o := range ec.orders {
		orders = append(orders, o)
	}
	return orders
}

// GetActiveOrders returns non-terminal orders
func (ec *EngagementCoordinator) GetActiveOrders() []*EngagementOrder {
	ec.ordersMu.RLock()
	defer ec.ordersMu.RUnlock()

	active := make([]*EngagementOrder, 0)
	for _, o := range ec.orders {
		switch o.State {
		case ENG_STATE_SUCCESS, ENG_STATE_FAILED, ENG_STATE_ABORTED:
			continue
		default:
			active = append(active, o)
		}
	}
	return active
}

// AuthorizeEngagement transitions order to authorized state
func (ec *EngagementCoordinator) AuthorizeEngagement(id string) error {
	ec.ordersMu.Lock()
	defer ec.ordersMu.Unlock()

	order, ok := ec.orders[id]
	if !ok {
		return ErrOrderNotFound
	}

	if order.State != ENG_STATE_PENDING {
		return ErrInvalidStateTransition
	}

	if ec.config.RequireNATApproval {
		order.State = ENG_STATE_AUTHORIZED
		order.UpdatedAt = time.Now()
	}

	return nil
}

// LaunchInterceptor transitions to launched state
func (ec *EngagementCoordinator) LaunchInterceptor(id string) error {
	ec.ordersMu.Lock()
	defer ec.ordersMu.Unlock()

	order, ok := ec.orders[id]
	if !ok {
		return ErrOrderNotFound
	}

	if order.State != ENG_STATE_AUTHORIZED && order.State != ENG_STATE_PENDING {
		return ErrInvalidStateTransition
	}

	// Update shooter status
	ec.shootersMu.Lock()
	if shooter, ok := ec.shooters[order.ShooterID]; ok {
		shooter.Status = SHOOTER_FIRING
		shooter.BatteryState.RoundsReady--
		shooter.BatteryState.EngagementActive = true
	}
	ec.shootersMu.Unlock()

	order.State = ENG_STATE_LAUNCHED
	order.UpdatedAt = time.Now()
	order.Attempts++

	return nil
}

// ReportIntercept updates state after intercept attempt
func (ec *EngagementCoordinator) ReportIntercept(id string, intercept bool) error {
	ec.ordersMu.Lock()
	defer ec.ordersMu.Unlock()

	order, ok := ec.orders[id]
	if !ok {
		return ErrOrderNotFound
	}

	if order.State != ENG_STATE_LAUNCHED {
		return ErrInvalidStateTransition
	}

	if intercept {
		order.State = ENG_STATE_INTERCEPT
	} else {
		// Check for retry
		if order.Attempts < order.MaxAttempts {
			order.State = ENG_STATE_AUTHORIZED // Can try again
		} else {
			order.State = ENG_STATE_FAILED
			order.AbortReason = "Max attempts exceeded"
		}
	}
	order.UpdatedAt = time.Now()

	return nil
}

// AssessKill updates state after kill assessment
func (ec *EngagementCoordinator) AssessKill(id string, level KillLevel, confidence float64) error {
	ec.ordersMu.Lock()
	defer ec.ordersMu.Unlock()

	order, ok := ec.orders[id]
	if !ok {
		return ErrOrderNotFound
	}

	if order.State != ENG_STATE_INTERCEPT && order.State != ENG_STATE_ASSESS {
		return ErrInvalidStateTransition
	}

	switch level {
	case KILL_LEVEL_CATK, KILL_LEVEL_NTK:
		order.State = ENG_STATE_SUCCESS
	case KILL_LEVEL_PKW, KILL_LEVEL_FAW:
		// Partial kill - might need another shot
		if order.Attempts < order.MaxAttempts {
			order.State = ENG_STATE_AUTHORIZED
		} else {
			order.State = ENG_STATE_FAILED
			order.AbortReason = "Partial kill, no rounds remaining"
		}
	default:
		order.State = ENG_STATE_FAILED
		order.AbortReason = "No kill"
	}
	order.UpdatedAt = time.Now()

	// Free up shooter
	ec.shootersMu.Lock()
	if shooter, ok := ec.shooters[order.ShooterID]; ok {
		shooter.Status = SHOOTER_AVAILABLE
		shooter.BatteryState.EngagementActive = false
	}
	ec.shootersMu.Unlock()

	return nil
}

// AbortEngagement aborts an engagement
func (ec *EngagementCoordinator) AbortEngagement(id string, reason string) error {
	ec.ordersMu.Lock()
	defer ec.ordersMu.Unlock()

	order, ok := ec.orders[id]
	if !ok {
		return ErrOrderNotFound
	}

	switch order.State {
	case ENG_STATE_SUCCESS, ENG_STATE_FAILED, ENG_STATE_ABORTED:
		return ErrInvalidStateTransition
	}

	order.State = ENG_STATE_ABORTED
	order.AbortReason = reason
	order.UpdatedAt = time.Now()

	// Free up shooter
	ec.shootersMu.Lock()
	if shooter, ok := ec.shooters[order.ShooterID]; ok {
		shooter.Status = SHOOTER_AVAILABLE
		shooter.BatteryState.EngagementActive = false
	}
	ec.shootersMu.Unlock()

	return nil
}

// CleanupOldOrders removes terminal orders older than timeout
func (ec *EngagementCoordinator) CleanupOldOrders(olderThan time.Duration) int {
	ec.ordersMu.Lock()
	defer ec.ordersMu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	count := 0

	for id, order := range ec.orders {
		if order.UpdatedAt.Before(cutoff) {
			switch order.State {
			case ENG_STATE_SUCCESS, ENG_STATE_FAILED, ENG_STATE_ABORTED:
				delete(ec.orders, id)
				count++
			}
		}
	}
	return count
}

// EngagementStats returns statistics
func (ec *EngagementCoordinator) EngagementStats() EngagementStats {
	ec.ordersMu.RLock()
	defer ec.ordersMu.RUnlock()

	var (
		active, pending, authorized, launched, success, failed, aborted int
	)
	for _, o := range ec.orders {
		switch o.State {
		case ENG_STATE_PENDING:
			pending++
			active++
		case ENG_STATE_AUTHORIZED:
			authorized++
			active++
		case ENG_STATE_LAUNCHED:
			launched++
			active++
		case ENG_STATE_INTERCEPT, ENG_STATE_ASSESS:
			active++
		case ENG_STATE_SUCCESS:
			success++
		case ENG_STATE_FAILED:
			failed++
		case ENG_STATE_ABORTED:
			aborted++
		}
	}

	return EngagementStats{
		ActiveEngagements: active,
		PendingOrders:    pending,
		AuthorizedOrders: authorized,
		LaunchedEngagements: launched,
		SuccessfulIntercepts: success,
		FailedIntercepts: failed,
		AbortedEngagements: aborted,
		TotalOrders: len(ec.orders),
	}
}

// EngagementStats holds coordinator statistics
type EngagementStats struct {
	ActiveEngagements    int `json:"active_engagements"`
	PendingOrders       int `json:"pending_orders"`
	AuthorizedOrders    int `json:"authorized_orders"`
	LaunchedEngagements int `json:"launched_engagements"`
	SuccessfulIntercepts int `json:"successful_intercepts"`
	FailedIntercepts    int `json:"failed_intercepts"`
	AbortedEngagements  int `json:"aborted_engagements"`
	TotalOrders         int `json:"total_orders"`
}

// Errors
type CoordinatorError int

const (
	ErrOrderNotFound CoordinatorError = iota
	ErrEngagementFull
	ErrInvalidStateTransition
	ErrShooterNotAvailable
)

func (e CoordinatorError) Error() string {
	switch e {
	case ErrOrderNotFound:
		return "engagement order not found"
	case ErrEngagementFull:
		return "engagement capacity reached"
	case ErrInvalidStateTransition:
		return "invalid state transition"
	case ErrShooterNotAvailable:
		return "shooter not available"
	default:
		return "unknown error"
	}
}

// EngagementReport for C2BMC / battle damage assessment
type EngagementReport struct {
	EngagementID   string          `json:"engagement_id"`
	TrackID         string          `json:"track_id"`
	State           EngagementState `json:"final_state"`
	Attempts        int             `json:"attempts"`
	KillLevel       KillLevel       `json:"kill_level"`
	KillConfidence  float64         `json:"kill_confidence"`
	InterceptTime   time.Time       `json:"intercept_time"`
	ReportTime      time.Time       `json:"report_time"`
	AbortReason     string          `json:"abort_reason,omitempty"`
}

// GenerateReport creates a report for an engagement
func (ec *EngagementCoordinator) GenerateReport(id string) (*EngagementReport, error) {
	order := ec.GetOrder(id)
	if order == nil {
		return nil, ErrOrderNotFound
	}

	return &EngagementReport{
		EngagementID: order.ID,
		TrackID:       order.TrackID,
		State:         order.State,
		Attempts:      order.Attempts,
		ReportTime:    time.Now(),
		AbortReason:   order.AbortReason,
	}, nil
}

// SelectShooter chooses best shooter for engagement
func (ec *EngagementCoordinator) SelectShooter(
	threatPos, threatVel [3]float64,
	threatType ThreatType,
	targetAlt, targetRange, timeToImpact float64,
	available []InterceptorType,
) (*Shooter, InterceptorType, error) {

	ec.shootersMu.RLock()
	defer ec.shootersMu.RUnlock()

	var bestShooter *Shooter
	var bestInterceptor InterceptorType
	bestProb := 0.0

	for _, shooter := range ec.shooters {
		if shooter.Status != SHOOTER_AVAILABLE {
			continue
		}
		if shooter.BatteryState.RoundsReady <= 0 {
			continue
		}

		// Check if shooter has suitable interceptors
		for _, intType := range available {
			// Check if shooter has this interceptor type
			hasType := false
			for _, sIntType := range shooter.Interceptors {
				if sIntType == intType {
					hasType = true
					break
				}
			}
			if !hasType {
				continue
			}

			// Check engagement zone
			ka := DefaultKillAssessment()
			zone := ka.EngagementZone(intType, targetAlt, targetRange, timeToImpact)
			if zone == "NO_ENGAGE" {
				continue
			}

			// Calculate intercept probability
			configs := DefaultInterceptorConfigs()
			config := configs[intType]
			prob := InterceptProbability(
				config,
				shooter.Location,
				[3]float64{0, 0, 0}, // Would need shooter velocity
				threatPos,
				threatVel,
				timeToImpact,
			)

			if prob > bestProb {
				bestProb = prob
				bestShooter = shooter
				bestInterceptor = intType
			}
		}
	}

	if bestShooter == nil {
		return nil, 0, ErrShooterNotAvailable
	}

	return bestShooter, bestInterceptor, nil
}

// SimulateEngagement runs a simulated engagement
func (ec *EngagementCoordinator) SimulateEngagement(
	threatPos, threatVel [3]float64,
	threatType ThreatType,
	shooterPos [3]float64,
	interceptorType InterceptorType,
) (bool, KillLevel, float64) {

	// Select interceptor
	png := DefaultPNG()

	// Initial interceptor velocity (toward threat)
	dirToTarget := [3]float64{
		threatPos[0] - shooterPos[0],
		threatPos[1] - shooterPos[1],
		threatPos[2] - shooterPos[2],
	}
	dist := math.Sqrt(dirToTarget[0]*dirToTarget[0]+dirToTarget[1]*dirToTarget[1]+dirToTarget[2]*dirToTarget[2])
	if dist > 0 {
		dirToTarget[0] /= dist
		dirToTarget[1] /= dist
		dirToTarget[2] /= dist
	}

	configs := DefaultInterceptorConfigs()
	config := configs[interceptorType]

	// Interceptor state
	intState := &InterceptorState{
		Type:      interceptorType,
		Position:  shooterPos,
		Velocity:  [3]float64{config.MaxVelocity * dirToTarget[0], config.MaxVelocity * dirToTarget[1], config.MaxVelocity * dirToTarget[2]},
	}

	// Simulate until intercept or miss
	var dt time.Duration = 100 * time.Millisecond
	for t := time.Duration(0); t < 120*time.Second; t += dt {
		// Time to intercept
		tti := TimeToIntercept(intState.Position, intState.Velocity, threatPos, threatVel)
		if tti < 0 {
			break
		}

		// Check for intercept (within lethal radius)
		rng := math.Sqrt(
			math.Pow(threatPos[0]-intState.Position[0], 2)+
				math.Pow(threatPos[1]-intState.Position[1], 2)+
				math.Pow(threatPos[2]-intState.Position[2], 2),
		)

		ka := DefaultKillAssessment()
		if rng < ka.blastRadius(threatType) {
			// Intercept - assess kill
			event := &InterceptEvent{
				InterceptPosition: intState.Position,
				TargetPosition:    threatPos,
				TargetVelocity:    threatVel,
				InterceptorType:   interceptorType,
				TargetType:        threatType,
			}
			level, conf := ka.Assess(event)
			return true, level, conf
		}

		// Update interceptor
		cmd := png.GuidanceCommand(intState, threatPos, threatVel, dt)
		intState.UpdateState(cmd, dt)

		// Update target position (simple constant velocity)
		threatPos[0] += threatVel[0] * dt.Seconds()
		threatPos[1] += threatVel[1] * dt.Seconds()
		threatPos[2] += threatVel[2] * dt.Seconds()
	}

	// No intercept
	return false, KILL_LEVEL_NONE, 0.0
}
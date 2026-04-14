// Package dis provides DIS exercise management
package dis

import (
	"sync"
	"time"
)

// ExerciseManager manages DIS exercise state
type ExerciseManager struct {
	config      *ExerciseConfig
	exerciseID  ExerciseID
	siteID      uint16
	applicationID uint16
	entities    map[uint16]map[uint16]map[uint16]EntityInfo // site.app.entity -> EntityInfo
	nextEntityID uint16
	mu          sync.RWMutex
	startTime   time.Time
}

// ExerciseConfig holds exercise configuration
type ExerciseConfig struct {
	ExerciseName     string `json:"exercise_name"`
	ExerciseID       ExerciseID `json:"exercise_id"`
	SiteID           uint16 `json:"site_id"`
	ApplicationID    uint16 `json:"application_id"`
	MaxEntities      int    `json:"max_entities"`
	AutoEntityID     bool   `json:"auto_entity_id"`
}

// ExerciseID represents a DIS exercise ID
type ExerciseID struct {
	SiteID       uint16 `json:"site_id"`
	ApplicationID uint16 `json:"application_id"`
	InstanceID   uint16 `json:"instance_id"`
}

// EntityInfo holds entity information
type EntityInfo struct {
	EntityID     EntityID `json:"entity_id"`
	EntityType   EntityType `json:"entity_type"`
	Force        ForceID `json:"force"`
	Marking      string `json:"marking"`
	LastUpdate   time.Time `json:"last_update"`
	IsActive     bool    `json:"is_active"`
}

// EntityID represents a DIS entity ID
type EntityID struct {
	SiteID       uint16 `json:"site_id"`
	ApplicationID uint16 `json:"application_id"`
	EntityNumber uint16 `json:"entity_number"`
}

// EntityType represents a DIS entity type
type EntityType struct {
	Kind        uint8 `json:"kind"`
	Domain      uint8 `json:"domain"`
	Country     uint16 `json:"country"`
	Category    uint8 `json:"category"`
	Subcategory uint8 `json:"subcategory"`
	Specific    uint8 `json:"specific"`
	Extra       uint8 `json:"extra"`
}

// ForceID represents a DIS force ID
type ForceID uint8

const (
	ForceOther ForceID = iota
	ForceFriendly
	ForceOpposing
	ForceNeutral
)

// DefaultExerciseConfig returns default exercise configuration
func DefaultExerciseConfig() *ExerciseConfig {
	return &ExerciseConfig{
		ExerciseName:  "VIGIL",
		ExerciseID:   ExerciseID{SiteID: 1, ApplicationID: 1, InstanceID: 1},
		SiteID:        1,
		ApplicationID: 1,
		MaxEntities:   10000,
		AutoEntityID:  true,
	}
}

// NewExerciseManager creates a new exercise manager
func NewExerciseManager(config *ExerciseConfig) *ExerciseManager {
	if config == nil {
		config = DefaultExerciseConfig()
	}
	
	return &ExerciseManager{
		config:        config,
		exerciseID:    config.ExerciseID,
		siteID:        config.SiteID,
		applicationID: config.ApplicationID,
		entities:      make(map[uint16]map[uint16]map[uint16]EntityInfo),
		nextEntityID:  1,
		startTime:     time.Now(),
	}
}

// StartExercise starts the exercise
func (em *ExerciseManager) StartExercise() error {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	em.startTime = time.Now()
	return nil
}

// StopExercise stops the exercise
func (em *ExerciseManager) StopExercise() error {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	// Clear all entities
	em.entities = make(map[uint16]map[uint16]map[uint16]EntityInfo)
	return nil
}

// AllocateEntityID allocates a new entity ID
func (em *ExerciseManager) AllocateEntityID() EntityID {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	// Use auto-allocation from same site/app
	id := EntityID{
		SiteID:       em.siteID,
		ApplicationID: em.applicationID,
		EntityNumber: em.nextEntityID,
	}
	
	em.nextEntityID++
	
	// Wrap around at max uint16
	if em.nextEntityID == 0 {
		em.nextEntityID = 1
	}
	
	return id
}

// RegisterEntity registers a new entity
func (em *ExerciseManager) RegisterEntity(entityID EntityID, entityType EntityType, force ForceID, marking string) error {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	// Initialize maps if needed
	if em.entities[entityID.SiteID] == nil {
		em.entities[entityID.SiteID] = make(map[uint16]map[uint16]EntityInfo)
	}
	if em.entities[entityID.SiteID][entityID.ApplicationID] == nil {
		em.entities[entityID.SiteID][entityID.ApplicationID] = make(map[uint16]EntityInfo)
	}
	
	em.entities[entityID.SiteID][entityID.ApplicationID][entityID.EntityNumber] = EntityInfo{
		EntityID:   entityID,
		EntityType: entityType,
		Force:      force,
		Marking:    marking,
		LastUpdate: time.Now(),
		IsActive:  true,
	}
	
	return nil
}

// UnregisterEntity unregisters an entity
func (em *ExerciseManager) UnregisterEntity(entityID EntityID) error {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	if em.entities[entityID.SiteID] == nil ||
		em.entities[entityID.SiteID][entityID.ApplicationID] == nil {
		return ErrEntityNotFound
	}
	
	delete(em.entities[entityID.SiteID][entityID.ApplicationID], entityID.EntityNumber)
	return nil
}

// GetEntity returns entity information
func (em *ExerciseManager) GetEntity(entityID EntityID) (EntityInfo, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	if em.entities[entityID.SiteID] == nil ||
		em.entities[entityID.SiteID][entityID.ApplicationID] == nil {
		return EntityInfo{}, ErrEntityNotFound
	}
	
	info, exists := em.entities[entityID.SiteID][entityID.ApplicationID][entityID.EntityNumber]
	if !exists {
		return EntityInfo{}, ErrEntityNotFound
	}
	
	return info, nil
}

// UpdateEntity updates entity information
func (em *ExerciseManager) UpdateEntity(entityID EntityID) error {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	if em.entities[entityID.SiteID] == nil ||
		em.entities[entityID.SiteID][entityID.ApplicationID] == nil {
		return ErrEntityNotFound
	}
	
	info, exists := em.entities[entityID.SiteID][entityID.ApplicationID][entityID.EntityNumber]
	if !exists {
		return ErrEntityNotFound
	}
	
	info.LastUpdate = time.Now()
	em.entities[entityID.SiteID][entityID.ApplicationID][entityID.EntityNumber] = info
	
	return nil
}

// GetAllEntities returns all entities
func (em *ExerciseManager) GetAllEntities() []EntityInfo {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	var entities []EntityInfo
	for _, apps := range em.entities {
		for _, entitiesMap := range apps {
			for _, info := range entitiesMap {
				entities = append(entities, info)
			}
		}
	}
	
	return entities
}

// GetEntitiesByForce returns entities by force
func (em *ExerciseManager) GetEntitiesByForce(force ForceID) []EntityInfo {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	var entities []EntityInfo
	for _, apps := range em.entities {
		for _, entitiesMap := range apps {
			for _, info := range entitiesMap {
				if info.Force == force {
					entities = append(entities, info)
				}
			}
		}
	}
	
	return entities
}

// GetEntityCount returns the count of registered entities
func (em *ExerciseManager) GetEntityCount() int {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	count := 0
	for _, apps := range em.entities {
		for _, entitiesMap := range apps {
			count += len(entitiesMap)
		}
	}
	
	return count
}

// CleanupStaleEntities removes entities not updated within duration
func (em *ExerciseManager) CleanupStaleEntities(maxAge time.Duration) int {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	count := 0
	
	for siteID, apps := range em.entities {
		for appID, entitiesMap := range apps {
			for entityNum, info := range entitiesMap {
				if info.LastUpdate.Before(cutoff) {
					delete(em.entities[siteID][appID], entityNum)
					count++
				}
			}
		}
	}
	
	return count
}

// GetExerciseTime returns exercise elapsed time
func (em *ExerciseManager) GetExerciseTime() time.Duration {
	return time.Since(em.startTime)
}

// GetExerciseID returns the exercise ID
func (em *ExerciseManager) GetExerciseID() ExerciseID {
	return em.exerciseID
}

// SetExerciseID sets the exercise ID
func (em *ExerciseManager) SetExerciseID(id ExerciseID) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.exerciseID = id
}

// GetSiteID returns the site ID
func (em *ExerciseManager) GetSiteID() uint16 {
	return em.siteID
}

// GetApplicationID returns the application ID
func (em *ExerciseManager) GetApplicationID() uint16 {
	return em.applicationID
}

// Stats returns exercise statistics
func (em *ExerciseManager) Stats() ExerciseStats {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	total := 0
	active := 0
	byForce := make(map[ForceID]int)
	
	for _, apps := range em.entities {
		for _, entitiesMap := range apps {
			for _, info := range entitiesMap {
				total++
				if info.IsActive {
					active++
				}
				byForce[info.Force]++
			}
		}
	}
	
	return ExerciseStats{
		ExerciseID:     em.exerciseID,
		SiteID:         em.siteID,
		ApplicationID:  em.applicationID,
		TotalEntities:  total,
		ActiveEntities: active,
		EntitiesByForce: byForce,
		ExerciseTime:   time.Since(em.startTime),
	}
}

// ExerciseStats holds exercise statistics
type ExerciseStats struct {
	ExerciseID      ExerciseID `json:"exercise_id"`
	SiteID          uint16 `json:"site_id"`
	ApplicationID   uint16 `json:"application_id"`
	TotalEntities   int    `json:"total_entities"`
	ActiveEntities  int    `json:"active_entities"`
	EntitiesByForce map[ForceID]int `json:"entities_by_force"`
	ExerciseTime    time.Duration `json:"exercise_time"`
}

// EntityIDEquals checks if two entity IDs are equal
func (e EntityID) Equals(other EntityID) bool {
	return e.SiteID == other.SiteID &&
		e.ApplicationID == other.ApplicationID &&
		e.EntityNumber == other.EntityNumber
}

// EntityIDLess checks if this entity ID is less than another
func (e EntityID) Less(other EntityID) bool {
	if e.SiteID != other.SiteID {
		return e.SiteID < other.SiteID
	}
	if e.ApplicationID != other.ApplicationID {
		return e.ApplicationID < other.ApplicationID
	}
	return e.EntityNumber < other.EntityNumber
}

// ToUint64 converts entity ID to uint64
func (e EntityID) ToUint64() uint64 {
	return uint64(e.SiteID)<<48 | uint64(e.ApplicationID)<<32 | uint64(e.EntityNumber)
}

// FromUint64 creates entity ID from uint64
func EntityIDFromUint64(v uint64) EntityID {
	return EntityID{
		SiteID:       uint16(v >> 48),
		ApplicationID: uint16(v >> 32),
		EntityNumber: uint16(v),
	}
}

// Errors
var (
	ErrEntityNotFound = &DISError{Code: "ENTITY_NOT_FOUND", Message: "entity not found"}
)
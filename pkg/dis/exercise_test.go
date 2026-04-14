package dis

import (
	"testing"
	"time"
)

// TestDefaultExerciseConfig tests default configuration
func TestDefaultExerciseConfig(t *testing.T) {
	config := DefaultExerciseConfig()
	
	if config.ExerciseName != "VIGIL" {
		t.Errorf("Expected exercise name VIGIL, got %s", config.ExerciseName)
	}
	if config.SiteID != 1 {
		t.Errorf("Expected site ID 1, got %d", config.SiteID)
	}
	if config.MaxEntities != 10000 {
		t.Errorf("Expected max entities 10000, got %d", config.MaxEntities)
	}
}

// TestNewExerciseManager tests exercise manager creation
func TestNewExerciseManager(t *testing.T) {
	em := NewExerciseManager(nil)
	
	if em == nil {
		t.Fatal("Exercise manager should not be nil")
	}
	
	if em.siteID != 1 {
		t.Error("Default site ID should be 1")
	}
}

// TestStartStopExercise tests exercise lifecycle
func TestStartStopExercise(t *testing.T) {
	em := NewExerciseManager(nil)
	
	err := em.StartExercise()
	if err != nil {
		t.Errorf("StartExercise failed: %v", err)
	}
	
	if em.startTime.IsZero() {
		t.Error("Start time should be set")
	}
	
	err = em.StopExercise()
	if err != nil {
		t.Errorf("StopExercise failed: %v", err)
	}
	
	if len(em.entities) != 0 {
		t.Error("Entities should be cleared after stop")
	}
}

// TestAllocateEntityID tests entity ID allocation
func TestAllocateEntityID(t *testing.T) {
	em := NewExerciseManager(nil)
	
	id1 := em.AllocateEntityID()
	if id1.EntityNumber != 1 {
		t.Errorf("First entity number should be 1, got %d", id1.EntityNumber)
	}
	
	id2 := em.AllocateEntityID()
	if id2.EntityNumber != 2 {
		t.Errorf("Second entity number should be 2, got %d", id2.EntityNumber)
	}
	
	// IDs should be from same site/app
	if id1.SiteID != em.siteID {
		t.Errorf("Site ID should be %d, got %d", em.siteID, id1.SiteID)
	}
}

// TestRegisterEntity tests entity registration
func TestRegisterEntity(t *testing.T) {
	em := NewExerciseManager(nil)
	
	entityID := em.AllocateEntityID()
	entityType := EntityType{
		Kind:     1, // Platform
		Domain:   2, // Air
		Country: 225, // USA
	}
	
	err := em.RegisterEntity(entityID, entityType, ForceFriendly, "F-15")
	if err != nil {
		t.Errorf("RegisterEntity failed: %v", err)
	}
	
	count := em.GetEntityCount()
	if count != 1 {
		t.Errorf("Expected 1 entity, got %d", count)
	}
}

// TestUnregisterEntity tests entity unregistration
func TestUnregisterEntity(t *testing.T) {
	em := NewExerciseManager(nil)
	
	entityID := em.AllocateEntityID()
	entityType := EntityType{Kind: 1, Domain: 2, Country: 225}
	
	em.RegisterEntity(entityID, entityType, ForceFriendly, "F-15")
	
	err := em.UnregisterEntity(entityID)
	if err != nil {
		t.Errorf("UnregisterEntity failed: %v", err)
	}
	
	count := em.GetEntityCount()
	if count != 0 {
		t.Errorf("Expected 0 entities, got %d", count)
	}
}

// TestUnregisterEntityNotFound tests unregistering non-existent entity
func TestUnregisterEntityNotFound(t *testing.T) {
	em := NewExerciseManager(nil)
	
	entityID := EntityID{SiteID: 999, ApplicationID: 999, EntityNumber: 999}
	
	err := em.UnregisterEntity(entityID)
	if err != ErrEntityNotFound {
		t.Errorf("Expected ErrEntityNotFound, got %v", err)
	}
}

// TestGetEntity tests entity retrieval
func TestGetEntity(t *testing.T) {
	em := NewExerciseManager(nil)
	
	entityID := em.AllocateEntityID()
	entityType := EntityType{Kind: 1, Domain: 2, Country: 225}
	
	em.RegisterEntity(entityID, entityType, ForceFriendly, "F-15")
	
	info, err := em.GetEntity(entityID)
	if err != nil {
		t.Errorf("GetEntity failed: %v", err)
	}
	
	if info.Marking != "F-15" {
		t.Errorf("Expected marking F-15, got %s", info.Marking)
	}
}

// TestGetEntityNotFound tests getting non-existent entity
func TestGetEntityNotFound(t *testing.T) {
	em := NewExerciseManager(nil)
	
	entityID := EntityID{SiteID: 999, ApplicationID: 999, EntityNumber: 999}
	
	_, err := em.GetEntity(entityID)
	if err != ErrEntityNotFound {
		t.Errorf("Expected ErrEntityNotFound, got %v", err)
	}
}

// TestUpdateEntity tests entity update
func TestUpdateEntity(t *testing.T) {
	em := NewExerciseManager(nil)
	
	entityID := em.AllocateEntityID()
	entityType := EntityType{Kind: 1, Domain: 2, Country: 225}
	
	em.RegisterEntity(entityID, entityType, ForceFriendly, "F-15")
	
	// Wait a bit
	time.Sleep(10 * time.Millisecond)
	
	err := em.UpdateEntity(entityID)
	if err != nil {
		t.Errorf("UpdateEntity failed: %v", err)
	}
	
	info, _ := em.GetEntity(entityID)
	
	// LastUpdate should be newer
	if info.LastUpdate.IsZero() {
		t.Error("LastUpdate should be set")
	}
}

// TestGetAllEntities tests getting all entities
func TestGetAllEntities(t *testing.T) {
	em := NewExerciseManager(nil)
	
	// Register multiple entities
	for i := 0; i < 5; i++ {
		entityID := em.AllocateEntityID()
		entityType := EntityType{Kind: 1, Domain: 2, Country: 225}
		em.RegisterEntity(entityID, entityType, ForceFriendly, "F-15")
	}
	
	entities := em.GetAllEntities()
	
	if len(entities) != 5 {
		t.Errorf("Expected 5 entities, got %d", len(entities))
	}
}

// TestGetEntitiesByForce tests filtering by force
func TestGetEntitiesByForce(t *testing.T) {
	em := NewExerciseManager(nil)
	
	// Register entities with different forces
	for i := 0; i < 3; i++ {
		entityID := em.AllocateEntityID()
		entityType := EntityType{Kind: 1, Domain: 2, Country: 225}
		em.RegisterEntity(entityID, entityType, ForceFriendly, "F-15")
	}
	
	for i := 0; i < 2; i++ {
		entityID := em.AllocateEntityID()
		entityType := EntityType{Kind: 1, Domain: 2, Country: 100}
		em.RegisterEntity(entityID, entityType, ForceOpposing, "MIG-29")
	}
	
	friendlies := em.GetEntitiesByForce(ForceFriendly)
	if len(friendlies) != 3 {
		t.Errorf("Expected 3 friendly entities, got %d", len(friendlies))
	}
	
	opposing := em.GetEntitiesByForce(ForceOpposing)
	if len(opposing) != 2 {
		t.Errorf("Expected 2 opposing entities, got %d", len(opposing))
	}
}

// TestCleanupStaleEntities tests stale entity cleanup
func TestCleanupStaleEntities(t *testing.T) {
	em := NewExerciseManager(nil)
	
	// Register entities
	for i := 0; i < 3; i++ {
		entityID := em.AllocateEntityID()
		entityType := EntityType{Kind: 1, Domain: 2, Country: 225}
		em.RegisterEntity(entityID, entityType, ForceFriendly, "F-15")
	}
	
	// Wait and clean
	time.Sleep(20 * time.Millisecond)
	removed := em.CleanupStaleEntities(10 * time.Millisecond)
	
	if removed != 3 {
		t.Errorf("Expected 3 entities removed, got %d", removed)
	}
	
	count := em.GetEntityCount()
	if count != 0 {
		t.Errorf("Expected 0 entities after cleanup, got %d", count)
	}
}

// TestExerciseTime tests exercise time
func TestExerciseTime(t *testing.T) {
	em := NewExerciseManager(nil)
	
	em.StartExercise()
	time.Sleep(50 * time.Millisecond)
	
	elapsed := em.GetExerciseTime()
	
	if elapsed < 50*time.Millisecond {
		t.Errorf("Exercise time should be at least 50ms, got %v", elapsed)
	}
}

// TestExerciseID tests exercise ID
func TestExerciseID(t *testing.T) {
	em := NewExerciseManager(nil)
	
	id := em.GetExerciseID()
	if id.SiteID != 1 {
		t.Errorf("Expected site ID 1, got %d", id.SiteID)
	}
	
	newID := ExerciseID{SiteID: 2, ApplicationID: 3, InstanceID: 4}
	em.SetExerciseID(newID)
	
	id = em.GetExerciseID()
	if id.SiteID != 2 {
		t.Errorf("Expected site ID 2, got %d", id.SiteID)
	}
}

// TestStats tests exercise statistics
func TestStats(t *testing.T) {
	em := NewExerciseManager(nil)
	
	// Register entities
	for i := 0; i < 3; i++ {
		entityID := em.AllocateEntityID()
		entityType := EntityType{Kind: 1, Domain: 2, Country: 225}
		em.RegisterEntity(entityID, entityType, ForceFriendly, "F-15")
	}
	
	stats := em.Stats()
	
	if stats.TotalEntities != 3 {
		t.Errorf("Expected 3 total entities, got %d", stats.TotalEntities)
	}
	
	if stats.ActiveEntities != 3 {
		t.Errorf("Expected 3 active entities, got %d", stats.ActiveEntities)
	}
	
	if stats.EntitiesByForce[ForceFriendly] != 3 {
		t.Errorf("Expected 3 friendly entities, got %d", stats.EntitiesByForce[ForceFriendly])
	}
}

// TestEntityIDEquivalence tests entity ID equality
func TestEntityIDEquivalence(t *testing.T) {
	id1 := EntityID{SiteID: 1, ApplicationID: 1, EntityNumber: 100}
	id2 := EntityID{SiteID: 1, ApplicationID: 1, EntityNumber: 100}
	id3 := EntityID{SiteID: 2, ApplicationID: 1, EntityNumber: 100}
	
	if !id1.Equals(id2) {
		t.Error("Equal IDs should be equal")
	}
	
	if id1.Equals(id3) {
		t.Error("Different IDs should not be equal")
	}
}

// TestEntityIDLess tests entity ID comparison
func TestEntityIDLess(t *testing.T) {
	id1 := EntityID{SiteID: 1, ApplicationID: 1, EntityNumber: 100}
	id2 := EntityID{SiteID: 1, ApplicationID: 1, EntityNumber: 200}
	id3 := EntityID{SiteID: 2, ApplicationID: 1, EntityNumber: 100}
	
	if !id1.Less(id2) {
		t.Error("id1 should be less than id2")
	}
	
	if !id1.Less(id3) {
		t.Error("id1 should be less than id3")
	}
	
	if id2.Less(id1) {
		t.Error("id2 should not be less than id1")
	}
}

// TestEntityIDConversion tests uint64 conversion
func TestEntityIDConversion(t *testing.T) {
	id := EntityID{SiteID: 1, ApplicationID: 2, EntityNumber: 3}
	
	v := id.ToUint64()
	
	id2 := EntityIDFromUint64(v)
	
	if !id.Equals(id2) {
		t.Errorf("Entity ID should survive round-trip: %v != %v", id, id2)
	}
}

// TestEntityIDWraparound tests entity ID wraparound
func TestEntityIDWraparound(t *testing.T) {
	em := NewExerciseManager(nil)
	
	// Allocate many IDs to test wraparound
	for i := 0; i < 65535; i++ {
		em.AllocateEntityID()
	}
	
	// Counter should have wrapped at this point
	// Just verify allocation still works
	id := em.AllocateEntityID()
	if id.EntityNumber == 0 {
		t.Error("Entity number should never be 0")
	}
}

// BenchmarkAllocateEntityID benchmarks entity ID allocation
func BenchmarkAllocateEntityID(b *testing.B) {
	em := NewExerciseManager(nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		em.AllocateEntityID()
	}
}

// BenchmarkRegisterEntity benchmarks entity registration
func BenchmarkRegisterEntity(b *testing.B) {
	em := NewExerciseManager(nil)
	entityType := EntityType{Kind: 1, Domain: 2, Country: 225}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entityID := em.AllocateEntityID()
		em.RegisterEntity(entityID, entityType, ForceFriendly, "F-15")
	}
}

// BenchmarkGetEntity benchmarks entity retrieval
func BenchmarkGetEntity(b *testing.B) {
	em := NewExerciseManager(nil)
	entityType := EntityType{Kind: 1, Domain: 2, Country: 225}
	
	// Register entity
	entityID := em.AllocateEntityID()
	em.RegisterEntity(entityID, entityType, ForceFriendly, "F-15")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		em.GetEntity(entityID)
	}
}
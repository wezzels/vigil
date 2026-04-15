package hla

import (
	"testing"
	"time"
)

// TestRTIType tests RTI type string representation
func TestRTIType(t *testing.T) {
	tests := []struct {
		rti  RTIType
		want string
	}{
		{RTIPortico, "Portico"},
		{RTIMak, "MAK"},
		{RTIPitch, "Pitch"},
	}

	for _, tt := range tests {
		if got := tt.rti.String(); got != tt.want {
			t.Errorf("RTIType(%d).String() = %s, want %s", tt.rti, got, tt.want)
		}
	}
}

// TestDefaultRTIConfig tests default RTI configuration
func TestDefaultRTIConfig(t *testing.T) {
	config := DefaultRTIConfig()

	if config.RTIType != RTIPortico {
		t.Errorf("Expected RTI type Portico, got %v", config.RTIType)
	}
	if config.FederationName != "VIGIL" {
		t.Errorf("Expected federation name VIGIL, got %s", config.FederationName)
	}
	if config.RTIPort != 8649 {
		t.Errorf("Expected RTI port 8649, got %d", config.RTIPort)
	}
}

// TestNewRTIAmbassador tests RTI ambassador creation
func TestNewRTIAmbassador(t *testing.T) {
	rti := NewRTIAmbassador(nil)

	if rti == nil {
		t.Fatal("RTI ambassador should not be nil")
	}

	if rti.IsConnected() {
		t.Error("RTI should not be connected initially")
	}
}

// TestCreateFederation tests federation creation
func TestCreateFederation(t *testing.T) {
	rti := NewRTIAmbassador(nil)

	err := rti.CreateFederation("TestFed", []string{})
	if err != nil {
		t.Errorf("CreateFederation failed: %v", err)
	}

	if !rti.IsConnected() {
		t.Error("RTI should be connected after creating federation")
	}

	// Creating again should fail
	err = rti.CreateFederation("TestFed2", []string{})
	if err != ErrFederationExists {
		t.Errorf("Expected ErrFederationExists, got %v", err)
	}
}

// TestDestroyFederation tests federation destruction
func TestDestroyFederation(t *testing.T) {
	rti := NewRTIAmbassador(nil)

	rti.CreateFederation("TestFed", []string{})

	err := rti.DestroyFederation("TestFed")
	if err != nil {
		t.Errorf("DestroyFederation failed: %v", err)
	}

	if rti.IsConnected() {
		t.Error("RTI should not be connected after destroying federation")
	}
}

// TestJoinFederation tests joining a federation
func TestJoinFederation(t *testing.T) {
	rti := NewRTIAmbassador(nil)

	err := rti.JoinFederation("TestFed", "federate1")
	if err != nil {
		t.Errorf("JoinFederation failed: %v", err)
	}

	if !rti.IsConnected() {
		t.Error("RTI should be connected after joining federation")
	}

	// Joining again should fail
	err = rti.JoinFederation("TestFed2", "federate2")
	if err != ErrAlreadyConnected {
		t.Errorf("Expected ErrAlreadyConnected, got %v", err)
	}
}

// TestResignFederation tests resigning from federation
func TestResignFederation(t *testing.T) {
	rti := NewRTIAmbassador(nil)

	rti.JoinFederation("TestFed", "federate1")

	err := rti.ResignFederation()
	if err != nil {
		t.Errorf("ResignFederation failed: %v", err)
	}

	if rti.IsConnected() {
		t.Error("RTI should not be connected after resigning")
	}
}

// TestRegisterObjectClass tests object class registration
func TestRegisterObjectClass(t *testing.T) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")

	handle, err := rti.RegisterObjectClass("BaseEntity")
	if err != nil {
		t.Errorf("RegisterObjectClass failed: %v", err)
	}

	if handle == 0 {
		t.Error("Object class handle should not be 0")
	}

	// Register another class
	handle2, err := rti.RegisterObjectClass("PhysicalEntity")
	if err != nil {
		t.Errorf("RegisterObjectClass failed: %v", err)
	}

	if handle2 == handle {
		t.Error("Object class handles should be unique")
	}
}

// TestRegisterObjectClassNotConnected tests registration when not connected
func TestRegisterObjectClassNotConnected(t *testing.T) {
	rti := NewRTIAmbassador(nil)

	_, err := rti.RegisterObjectClass("BaseEntity")
	if err != ErrNotConnected {
		t.Errorf("Expected ErrNotConnected, got %v", err)
	}
}

// TestPublishObjectClass tests publishing object class
func TestPublishObjectClass(t *testing.T) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")
	handle, _ := rti.RegisterObjectClass("BaseEntity")

	attrs := []AttributeHandle{1, 2, 3}
	err := rti.PublishObjectClass(handle, attrs)
	if err != nil {
		t.Errorf("PublishObjectClass failed: %v", err)
	}
}

// TestSubscribeObjectClass tests subscribing to object class
func TestSubscribeObjectClass(t *testing.T) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")
	handle, _ := rti.RegisterObjectClass("BaseEntity")

	attrs := []AttributeHandle{1, 2, 3}
	err := rti.SubscribeObjectClass(handle, attrs)
	if err != nil {
		t.Errorf("SubscribeObjectClass failed: %v", err)
	}
}

// TestRegisterObjectInstance tests object instance registration
func TestRegisterObjectInstance(t *testing.T) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")
	classHandle, _ := rti.RegisterObjectClass("BaseEntity")

	instance, err := rti.RegisterObjectInstance(classHandle)
	if err != nil {
		t.Errorf("RegisterObjectInstance failed: %v", err)
	}

	if instance == 0 {
		t.Error("Instance handle should not be 0")
	}
}

// TestUpdateAttributeValues tests attribute update
func TestUpdateAttributeValues(t *testing.T) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")
	classHandle, _ := rti.RegisterObjectClass("BaseEntity")
	instance, _ := rti.RegisterObjectInstance(classHandle)

	values := map[AttributeHandle][]byte{
		1: []byte("position"),
		2: []byte("velocity"),
	}

	err := rti.UpdateAttributeValues(instance, values)
	if err != nil {
		t.Errorf("UpdateAttributeValues failed: %v", err)
	}
}

// TestDeleteObjectInstance tests object instance deletion
func TestDeleteObjectInstance(t *testing.T) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")
	classHandle, _ := rti.RegisterObjectClass("BaseEntity")
	instance, _ := rti.RegisterObjectInstance(classHandle)

	err := rti.DeleteObjectInstance(instance)
	if err != nil {
		t.Errorf("DeleteObjectInstance failed: %v", err)
	}
}

// TestRegisterInteractionClass tests interaction class registration
func TestRegisterInteractionClass(t *testing.T) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")

	handle, err := rti.RegisterInteractionClass("WeaponFire")
	if err != nil {
		t.Errorf("RegisterInteractionClass failed: %v", err)
	}

	if handle == 0 {
		t.Error("Interaction class handle should not be 0")
	}
}

// TestSendInteraction tests sending interaction
func TestSendInteraction(t *testing.T) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")
	handle, _ := rti.RegisterInteractionClass("WeaponFire")

	params := map[ParameterHandle][]byte{
		1: []byte("target"),
		2: []byte("munition"),
	}

	err := rti.SendInteraction(handle, params)
	if err != nil {
		t.Errorf("SendInteraction failed: %v", err)
	}
}

// TestTimeRegulation tests time regulation
func TestTimeRegulation(t *testing.T) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")

	err := rti.EnableTimeRegulation(1 * time.Second)
	if err != nil {
		t.Errorf("EnableTimeRegulation failed: %v", err)
	}

	stats := rti.Stats()
	if !stats.TimeRegulated {
		t.Error("Time regulation should be enabled")
	}

	err = rti.DisableTimeRegulation()
	if err != nil {
		t.Errorf("DisableTimeRegulation failed: %v", err)
	}

	stats = rti.Stats()
	if stats.TimeRegulated {
		t.Error("Time regulation should be disabled")
	}
}

// TestTimeConstrained tests time constrained mode
func TestTimeConstrained(t *testing.T) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")

	err := rti.EnableTimeConstrained()
	if err != nil {
		t.Errorf("EnableTimeConstrained failed: %v", err)
	}

	stats := rti.Stats()
	if !stats.TimeConstrained {
		t.Error("Time constrained should be enabled")
	}

	err = rti.DisableTimeConstrained()
	if err != nil {
		t.Errorf("DisableTimeConstrained failed: %v", err)
	}

	stats = rti.Stats()
	if stats.TimeConstrained {
		t.Error("Time constrained should be disabled")
	}
}

// TestTimeAdvance tests time advance
func TestTimeAdvance(t *testing.T) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")

	targetTime := time.Now().Add(10 * time.Second)
	err := rti.TimeAdvanceRequest(targetTime)
	if err != nil {
		t.Errorf("TimeAdvanceRequest failed: %v", err)
	}

	currentTime, err := rti.QueryFederateTime()
	if err != nil {
		t.Errorf("QueryFederateTime failed: %v", err)
	}

	if !currentTime.Equal(targetTime) {
		t.Errorf("Federate time should be %v, got %v", targetTime, currentTime)
	}
}

// TestSynchronizationPoint tests sync point registration
func TestSynchronizationPoint(t *testing.T) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")

	err := rti.RegisterFederationSynchronizationPoint("ReadyToStart", "")
	if err != nil {
		t.Errorf("RegisterFederationSynchronizationPoint failed: %v", err)
	}

	err = rti.AchieveSynchronizationPoint("ReadyToStart")
	if err != nil {
		t.Errorf("AchieveSynchronizationPoint failed: %v", err)
	}
}

// TestStats tests RTI statistics
func TestStats(t *testing.T) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")
	rti.RegisterObjectClass("BaseEntity")
	rti.RegisterInteractionClass("WeaponFire")

	stats := rti.Stats()

	if !stats.Connected {
		t.Error("Should be connected")
	}
	if stats.FederationName != "TestFed" {
		t.Errorf("Federation name should be TestFed, got %s", stats.FederationName)
	}
	if stats.FederateName != "federate1" {
		t.Errorf("Federate name should be federate1, got %s", stats.FederateName)
	}
	if stats.ObjectClasses != 1 {
		t.Errorf("Should have 1 object class, got %d", stats.ObjectClasses)
	}
	if stats.Interactions != 1 {
		t.Errorf("Should have 1 interaction, got %d", stats.Interactions)
	}
}

// TestShutdown tests RTI shutdown
func TestShutdown(t *testing.T) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")

	err := rti.Shutdown()
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	if rti.IsConnected() {
		t.Error("Should not be connected after shutdown")
	}
}

// TestRTIError tests RTI error
func TestRTIError(t *testing.T) {
	err := ErrNotConnected

	if err.Code != "NOT_CONNECTED" {
		t.Errorf("Error code should be NOT_CONNECTED, got %s", err.Code)
	}

	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

// TestNotConnectedErrors tests all not connected errors
func TestNotConnectedErrors(t *testing.T) {
	rti := NewRTIAmbassador(nil)

	// All operations should fail when not connected
	if _, err := rti.RegisterObjectClass("Test"); err != ErrNotConnected {
		t.Errorf("Expected ErrNotConnected, got %v", err)
	}

	if err := rti.PublishObjectClass(1, nil); err != ErrNotConnected {
		t.Errorf("Expected ErrNotConnected, got %v", err)
	}

	if err := rti.SubscribeObjectClass(1, nil); err != ErrNotConnected {
		t.Errorf("Expected ErrNotConnected, got %v", err)
	}

	if _, err := rti.RegisterObjectInstance(1); err != ErrNotConnected {
		t.Errorf("Expected ErrNotConnected, got %v", err)
	}

	if err := rti.EnableTimeRegulation(time.Second); err != ErrNotConnected {
		t.Errorf("Expected ErrNotConnected, got %v", err)
	}

	if err := rti.EnableTimeConstrained(); err != ErrNotConnected {
		t.Errorf("Expected ErrNotConnected, got %v", err)
	}
}

// BenchmarkRegisterObjectClass benchmarks object class registration
func BenchmarkRegisterObjectClass(b *testing.B) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rti.RegisterObjectClass("TestClass")
	}
}

// BenchmarkSendInteraction benchmarks interaction sending
func BenchmarkSendInteraction(b *testing.B) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")
	handle, _ := rti.RegisterInteractionClass("TestInteraction")

	params := map[ParameterHandle][]byte{
		1: []byte("test"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rti.SendInteraction(handle, params)
	}
}

// BenchmarkTimeAdvance benchmarks time advance
func BenchmarkTimeAdvance(b *testing.B) {
	rti := NewRTIAmbassador(nil)
	rti.JoinFederation("TestFed", "federate1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rti.TimeAdvanceRequest(time.Now().Add(time.Duration(i) * time.Millisecond))
	}
}

package link16

import (
	"testing"
	"time"
)

// TestJ120Parser tests parser creation
func TestJ120Parser(t *testing.T) {
	parser := NewJ120Parser()
	
	if parser == nil {
		t.Fatal("Parser should not be nil")
	}
}

// TestJ120MessageParse tests message parsing
func TestJ120MessageParse(t *testing.T) {
	parser := NewJ120Parser()
	
	// Create test words
	words := []uint32{
		0x00010002, // Track number 1, Mission ID 2
		(3 << 24) | (1 << 20) | (5 << 16) | 0x2D00, // Mission STRIKE, status ASSIGNED, priority 5
		(10000 << 16) | 100, // Altitude 10000m, Unit 100
		(3 << 24) | (60 << 8) | 120, // Force FRIEND, 60 min start, 120 min end
	}
	
	msg, err := parser.Parse(words)
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}
	
	if msg.TrackNumber != 1 {
		t.Errorf("Track number should be 1, got %d", msg.TrackNumber)
	}
	
	if msg.MissionID != 2 {
		t.Errorf("Mission ID should be 2, got %d", msg.MissionID)
	}
}

// TestJ120MessageParseTooShort tests parsing with insufficient words
func TestJ120MessageParseTooShort(t *testing.T) {
	parser := NewJ120Parser()
	
	words := []uint32{0x00010002, 0x00000000}
	
	_, err := parser.Parse(words)
	if err == nil {
		t.Error("Expected error for insufficient words")
	}
}

// TestJ120MessageSerialize tests message serialization
func TestJ120MessageSerialize(t *testing.T) {
	parser := NewJ120Parser()
	
	msg := &J120Message{
		TrackNumber:      100,
		MissionID:        5,
		MissionType:      J120MissionStrike,
		MissionStatus:     J120StatusActive,
		Priority:         7,
		TargetLatitude:    45.0,
		TargetLongitude:   -120.0,
		TargetAltitude:   5000.0,
		AssignedUnit:      200,
		AssignedForce:     IdentityFriend,
		AssignmentStatus: J120AssignmentAccepted,
		StartTime:         time.Now(),
		EndTime:           time.Now().Add(2 * time.Hour),
		Time:              time.Now(),
	}
	
	words := parser.Serialize(msg)
	
	if len(words) != J120WordCount {
		t.Errorf("Expected %d words, got %d", J120WordCount, len(words))
	}
}

// TestJ120Roundtrip tests parse/serialize roundtrip
func TestJ120Roundtrip(t *testing.T) {
	parser := NewJ120Parser()
	
	original := &J120Message{
		TrackNumber:      500,
		MissionID:        10,
		MissionType:      J120MissionCAP,
		MissionStatus:     J120StatusAssigned,
		Priority:         8,
		TargetLatitude:    33.5,
		TargetLongitude:   -117.0,
		TargetAltitude:   10000.0,
		AssignedUnit:      50,
		AssignedForce:     IdentityFriend,
		AssignmentStatus: J120AssignmentAccepted,
		StartTime:         time.Now(),
		EndTime:           time.Now().Add(4 * time.Hour),
		Time:              time.Now(),
	}
	
	words := parser.Serialize(original)
	parsed, err := parser.Parse(words)
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}
	
	if parsed.TrackNumber != original.TrackNumber {
		t.Errorf("Track number mismatch: got %d, want %d",
			parsed.TrackNumber, original.TrackNumber)
	}
	
	if parsed.MissionID != original.MissionID {
		t.Errorf("Mission ID mismatch: got %d, want %d",
			parsed.MissionID, original.MissionID)
	}
	
	if parsed.MissionType != original.MissionType {
		t.Errorf("Mission type mismatch: got %d, want %d",
			parsed.MissionType, original.MissionType)
	}
}

// TestMissionTypeStrings tests mission type string conversion
func TestMissionTypeStrings(t *testing.T) {
	tests := []struct {
		missionType uint8
		expected    string
	}{
		{J120MissionCAP, "CAP"},
		{J120MissionStrike, "STRIKE"},
		{J120MissionSEAD, "SEAD"},
		{J120MissionCAS, "CAS"},
		{J120MissionRecon, "RECON"},
		{J120MissionCSAR, "CSAR"},
		{J120MissionAWACS, "AWACS"},
		{J120MissionTanker, "TANKER"},
		{J120MissionEW, "EW"},
		{J120MissionTraining, "TRAINING"},
		{J120MissionTest, "TEST"},
		{J120MissionExercise, "EXERCISE"},
		{99, "UNKNOWN"},
	}
	
	for _, tt := range tests {
		result := GetMissionTypeString(tt.missionType)
		if result != tt.expected {
			t.Errorf("GetMissionTypeString(%d) = %s, want %s",
				tt.missionType, result, tt.expected)
		}
	}
}

// TestMissionStatusStrings tests mission status string conversion
func TestMissionStatusStrings(t *testing.T) {
	tests := []struct {
		status   uint8
		expected string
	}{
		{J120StatusPlanned, "PLANNED"},
		{J120StatusAssigned, "ASSIGNED"},
		{J120StatusActive, "ACTIVE"},
		{J120StatusComplete, "COMPLETE"},
		{J120StatusCancelled, "CANCELLED"},
		{J120StatusAborted, "ABORTED"},
		{J120StatusDelayed, "DELAYED"},
		{J120StatusSuspended, "SUSPENDED"},
		{99, "UNKNOWN"},
	}
	
	for _, tt := range tests {
		result := GetMissionStatusString(tt.status)
		if result != tt.expected {
			t.Errorf("GetMissionStatusString(%d) = %s, want %s",
				tt.status, result, tt.expected)
		}
	}
}

// TestAssignmentStatusStrings tests assignment status string conversion
func TestAssignmentStatusStrings(t *testing.T) {
	tests := []struct {
		status   uint8
		expected string
	}{
		{J120AssignmentPending, "PENDING"},
		{J120AssignmentAccepted, "ACCEPTED"},
		{J120AssignmentRejected, "REJECTED"},
		{J120AssignmentComplete, "COMPLETE"},
		{J120AssignmentFailed, "FAILED"},
		{99, "UNKNOWN"},
	}
	
	for _, tt := range tests {
		result := GetAssignmentStatusString(tt.status)
		if result != tt.expected {
			t.Errorf("GetAssignmentStatusString(%d) = %s, want %s",
				tt.status, result, tt.expected)
		}
	}
}

// TestJ120Builder tests message builder
func TestJ120Builder(t *testing.T) {
	builder := NewJ120Builder()
	
	msg := builder.
		SetTrackNumber(100).
		SetMissionID(5).
		SetMissionType(J120MissionCAP).
		SetStatus(J120StatusActive).
		SetPriority(10).
		SetTarget(33.5, -117.0, 5000).
		SetAssignedUnit(200, IdentityFriend).
		SetAssignmentStatus(J120AssignmentAccepted).
		SetTimes(time.Now(), time.Now().Add(2*time.Hour)).
		Build()
	
	if msg.TrackNumber != 100 {
		t.Errorf("Track number should be 100, got %d", msg.TrackNumber)
	}
	
	if msg.MissionType != J120MissionCAP {
		t.Errorf("Mission type should be CAP, got %d", msg.MissionType)
	}
	
	if msg.TargetLatitude != 33.5 {
		t.Errorf("Latitude should be 33.5, got %f", msg.TargetLatitude)
	}
}

// TestJ120MissionConversion tests mission conversion
func TestJ120MissionConversion(t *testing.T) {
	msg := &J120Message{
		TrackNumber:      100,
		MissionID:        5,
		MissionType:      J120MissionStrike,
		MissionStatus:     J120StatusActive,
		TargetLatitude:    45.0,
		TargetLongitude:   -120.0,
		TargetAltitude:   10000.0,
		AssignedUnit:      200,
		AssignedForce:     IdentityFriend,
		AssignmentStatus: J120AssignmentAccepted,
		Priority:         7,
		Time:              time.Now(),
	}
	
	mission := msg.ToMission()
	
	if mission.MissionID != 5 {
		t.Errorf("Mission ID should be 5, got %d", mission.MissionID)
	}
	
	if mission.MissionType != "STRIKE" {
		t.Errorf("Mission type should be STRIKE, got %s", mission.MissionType)
	}
	
	if mission.TargetPosition[0] != 45.0 {
		t.Errorf("Latitude should be 45.0, got %f", mission.TargetPosition[0])
	}
}

// TestJ120Constants tests constant values
func TestJ120Constants(t *testing.T) {
	if J120WordCount != 4 {
		t.Errorf("J120WordCount should be 4, got %d", J120WordCount)
	}
	
	if J120MissionCAP != 1 {
		t.Errorf("J120MissionCAP should be 1, got %d", J120MissionCAP)
	}
	
	if J120StatusActive != 2 {
		t.Errorf("J120StatusActive should be 2, got %d", J120StatusActive)
	}
}

// BenchmarkJ120Parse benchmarks J12.0 parsing
func BenchmarkJ120Parse(b *testing.B) {
	parser := NewJ120Parser()
	words := []uint32{0x00010002, 0x03150000, 0x27100064, 0x36000078}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(words)
	}
}

// BenchmarkJ120Serialize benchmarks J12.0 serialization
func BenchmarkJ120Serialize(b *testing.B) {
	parser := NewJ120Parser()
	msg := &J120Message{
		TrackNumber:      100,
		MissionID:        5,
		MissionType:      J120MissionStrike,
		MissionStatus:     J120StatusActive,
		Priority:         7,
		TargetLatitude:    45.0,
		TargetLongitude:   -120.0,
		TargetAltitude:   10000.0,
		AssignedUnit:      200,
		AssignedForce:     IdentityFriend,
		AssignmentStatus: J120AssignmentAccepted,
		Time:              time.Now(),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Serialize(msg)
	}
}

// BenchmarkJ120Builder benchmarks J12.0 builder
func BenchmarkJ120Builder(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewJ120Builder().
			SetTrackNumber(100).
			SetMissionID(5).
			SetMissionType(J120MissionCAP).
			SetStatus(J120StatusActive).
			SetPriority(10).
			SetTarget(33.5, -117.0, 5000).
			Build()
	}
}

// BenchmarkJ120MissionConversion benchmarks mission conversion
func BenchmarkJ120MissionConversion(b *testing.B) {
	msg := &J120Message{
		TrackNumber:      100,
		MissionID:        5,
		MissionType:      J120MissionStrike,
		MissionStatus:     J120StatusActive,
		TargetLatitude:    45.0,
		TargetLongitude:   -120.0,
		TargetAltitude:   10000.0,
		AssignedUnit:      200,
		AssignedForce:     IdentityFriend,
		AssignmentStatus: J120AssignmentAccepted,
		Time:              time.Now(),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.ToMission()
	}
}
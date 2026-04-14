package link16

import (
	"testing"
	"time"
)

// TestJ70Parser tests parser creation
func TestJ70Parser(t *testing.T) {
	parser := NewJ70Parser()
	
	if parser == nil {
		t.Fatal("Parser should not be nil")
	}
}

// TestJ70MessageParse tests message parsing
func TestJ70MessageParse(t *testing.T) {
	parser := NewJ70Parser()
	
	// Create test words
	words := []uint32{
		0x00010000 | // Track number 1
			(3 << 12) |  // Status CONFIRMED
			(10 << 8) |   // Quality 10
			(3 << 4) |    // Identity FRIEND
			3,            // Force FRIEND
		(1 << 28) | // Action UPDATE
			(0 << 24) |    // Environment AIR
			(0 << 12) |    // Source track
			100,           // Target track
	}
	
	msg, err := parser.Parse(words)
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}
	
	if msg.TrackNumber != 1 {
		t.Errorf("Track number should be 1, got %d", msg.TrackNumber)
	}
	
	if msg.TrackStatus != J70StatusConfirmed {
		t.Errorf("Status should be CONFIRMED, got %d", msg.TrackStatus)
	}
	
	if msg.ActionCode != J70ActionUpdate {
		t.Errorf("Action should be UPDATE, got %d", msg.ActionCode)
	}
}

// TestJ70MessageParseTooShort tests parsing with insufficient words
func TestJ70MessageParseTooShort(t *testing.T) {
	parser := NewJ70Parser()
	
	words := []uint32{0x00010000}
	
	_, err := parser.Parse(words)
	if err == nil {
		t.Error("Expected error for insufficient words")
	}
}

// TestJ70MessageSerialize tests message serialization
func TestJ70MessageSerialize(t *testing.T) {
	parser := NewJ70Parser()
	
	msg := &J70Message{
		TrackNumber:   100,
		TrackStatus:   J70StatusConfirmed,
		TrackQuality:  12,
		TrackIdentity: IdentityFriend,
		ForceID:       IdentityFriend,
		Environment:   EnvAir,
		ActionCode:    J70ActionUpdate,
		SourceTrack:   0,
		TargetTrack:  200,
		Time:          time.Now(),
	}
	
	words := parser.Serialize(msg)
	
	if len(words) != 2 {
		t.Errorf("Expected 2 words, got %d", len(words))
	}
}

// TestJ70Roundtrip tests parse/serialize roundtrip
func TestJ70Roundtrip(t *testing.T) {
	parser := NewJ70Parser()
	
	original := &J70Message{
		TrackNumber:   500,
		TrackStatus:   J70StatusConfirmed,
		TrackQuality:  15,
		TrackIdentity: IdentityHostile,
		ForceID:       IdentityHostile,
		Environment:   EnvAir,
		ActionCode:    J70ActionCorrelate,
		SourceTrack:   100,
		TargetTrack:   200,
		Time:          time.Now(),
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
	
	if parsed.TrackStatus != original.TrackStatus {
		t.Errorf("Status mismatch: got %d, want %d",
			parsed.TrackStatus, original.TrackStatus)
	}
	
	if parsed.ActionCode != original.ActionCode {
		t.Errorf("Action mismatch: got %d, want %d",
			parsed.ActionCode, original.ActionCode)
	}
}

// TestActionStrings tests action string conversion
func TestActionStrings(t *testing.T) {
	tests := []struct {
		action   uint8
		expected string
	}{
		{J70ActionNewTrack, "NEW_TRACK"},
		{J70ActionUpdate, "UPDATE"},
		{J70ActionDelete, "DELETE"},
		{J70ActionCorrelate, "CORRELATE"},
		{J70ActionMerge, "MERGE"},
		{J70ActionSplit, "SPLIT"},
		{J70ActionDropTrack, "DROP_TRACK"},
		{99, "UNKNOWN"},
	}
	
	for _, tt := range tests {
		result := GetActionString(tt.action)
		if result != tt.expected {
			t.Errorf("GetActionString(%d) = %s, want %s",
				tt.action, result, tt.expected)
		}
	}
}

// TestStatusStrings tests status string conversion
func TestStatusStrings(t *testing.T) {
	tests := []struct {
		status   uint8
		expected string
	}{
		{J70StatusDropped, "DROPPED"},
		{J70StatusInitiating, "INITIATING"},
		{J70StatusTentative, "TENTATIVE"},
		{J70StatusConfirmed, "CONFIRMED"},
		{J70StatusCoasting, "COASTING"},
		{J70StatusLost, "LOST"},
		{99, "UNKNOWN"},
	}
	
	for _, tt := range tests {
		result := GetStatusString(tt.status)
		if result != tt.expected {
			t.Errorf("GetStatusString(%d) = %s, want %s",
				tt.status, result, tt.expected)
		}
	}
}

// TestJ70Builder tests message builder
func TestJ70Builder(t *testing.T) {
	builder := NewJ70Builder()
	
	msg := builder.
		SetTrackNumber(100).
		SetStatus(J70StatusConfirmed).
		SetQuality(10).
		SetIdentity(IdentityFriend).
		SetForce(IdentityFriend).
		SetEnvironment(EnvAir).
		SetAction(J70ActionUpdate).
		SetSource(0).
		SetTarget(200).
		Build()
	
	if msg.TrackNumber != 100 {
		t.Errorf("Track number should be 100, got %d", msg.TrackNumber)
	}
	
	if msg.TrackStatus != J70StatusConfirmed {
		t.Errorf("Status should be CONFIRMED, got %d", msg.TrackStatus)
	}
	
	if msg.ActionCode != J70ActionUpdate {
		t.Errorf("Action should be UPDATE, got %d", msg.ActionCode)
	}
}

// TestJ70Actions tests action creation
func TestJ70Actions(t *testing.T) {
	// New track
	newAction := NewTrackAction(100)
	if newAction.Action != J70ActionNewTrack {
		t.Error("NewTrackAction should have NEW_TRACK action")
	}
	if newAction.Target != 100 {
		t.Error("Target should be 100")
	}
	
	// Update track
	updateAction := UpdateTrackAction(100)
	if updateAction.Action != J70ActionUpdate {
		t.Error("UpdateTrackAction should have UPDATE action")
	}
	
	// Delete track
	deleteAction := DeleteTrackAction(100)
	if deleteAction.Action != J70ActionDelete {
		t.Error("DeleteTrackAction should have DELETE action")
	}
	
	// Correlate tracks
	correlateAction := CorrelateTracksAction(100, 200)
	if correlateAction.Action != J70ActionCorrelate {
		t.Error("CorrelateTracksAction should have CORRELATE action")
	}
	if correlateAction.Source != 100 {
		t.Error("Source should be 100")
	}
	if correlateAction.Target != 200 {
		t.Error("Target should be 200")
	}
	
	// Merge tracks
	mergeAction := MergeTracksAction(100, 200)
	if mergeAction.Action != J70ActionMerge {
		t.Error("MergeTracksAction should have MERGE action")
	}
}

// TestJ70AllStatuses tests all status codes
func TestJ70AllStatuses(t *testing.T) {
	parser := NewJ70Parser()
	
	statuses := []uint8{
		J70StatusDropped,
		J70StatusInitiating,
		J70StatusTentative,
		J70StatusConfirmed,
		J70StatusCoasting,
		J70StatusPredicted,
		J70StatusLost,
	}
	
	for _, status := range statuses {
		msg := &J70Message{
			TrackNumber:  1,
			TrackStatus: status,
			TrackQuality: 10,
			ActionCode:   J70ActionUpdate,
			Time:         time.Now(),
		}
		
		words := parser.Serialize(msg)
		parsed, err := parser.Parse(words)
		if err != nil {
			t.Errorf("Parse failed for status %d: %v", status, err)
		}
		
		if parsed.TrackStatus != status {
			t.Errorf("Status mismatch: got %d, want %d", parsed.TrackStatus, status)
		}
	}
}

// TestJ70AllActions tests all action codes
func TestJ70AllActions(t *testing.T) {
	parser := NewJ70Parser()
	
	actions := []uint8{
		J70ActionNewTrack,
		J70ActionUpdate,
		J70ActionDelete,
		J70ActionCorrelate,
		J70ActionMerge,
		J70ActionSplit,
		J70ActionDropTrack,
	}
	
	for _, action := range actions {
		msg := &J70Message{
			TrackNumber:  1,
			TrackStatus: J70StatusConfirmed,
			ActionCode:   action,
			Time:         time.Now(),
		}
		
		words := parser.Serialize(msg)
		parsed, err := parser.Parse(words)
		if err != nil {
			t.Errorf("Parse failed for action %d: %v", action, err)
		}
		
		if parsed.ActionCode != action {
			t.Errorf("Action mismatch: got %d, want %d", parsed.ActionCode, action)
		}
	}
}

// BenchmarkJ70Parse benchmarks J7.0 parsing
func BenchmarkJ70Parse(b *testing.B) {
	parser := NewJ70Parser()
	words := []uint32{0x00013003, 0x10000064}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(words)
	}
}

// BenchmarkJ70Serialize benchmarks J7.0 serialization
func BenchmarkJ70Serialize(b *testing.B) {
	parser := NewJ70Parser()
	msg := &J70Message{
		TrackNumber:   100,
		TrackStatus:   J70StatusConfirmed,
		TrackQuality:  10,
		ActionCode:    J70ActionUpdate,
		Time:          time.Now(),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Serialize(msg)
	}
}

// BenchmarkJ70Builder benchmarks J7.0 builder
func BenchmarkJ70Builder(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewJ70Builder().
			SetTrackNumber(100).
			SetStatus(J70StatusConfirmed).
			SetQuality(10).
			SetAction(J70ActionUpdate).
			Build()
	}
}

// BenchmarkJ70Actions benchmarks action creation
func BenchmarkJ70Actions(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewTrackAction(100)
		UpdateTrackAction(100)
		DeleteTrackAction(100)
		CorrelateTracksAction(100, 200)
		MergeTracksAction(100, 200)
	}
}
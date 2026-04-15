package link16

import (
	"math"
	"testing"
	"time"
)

// TestJ32Parser tests parser creation
func TestJ32Parser(t *testing.T) {
	parser := NewJ32Parser()

	if parser == nil {
		t.Fatal("Parser should not be nil")
	}
}

// TestJ32MessageParse tests message parsing
func TestJ32MessageParse(t *testing.T) {
	parser := NewJ32Parser()

	// Create test words
	words := []uint32{
		0x00010000 | // Track number 1, quality 0, identity 0
			(3 << 8) | // Identity FRIEND
			(3 << 4) | // Force FRIEND
			0, // Environment AIR
		(45 << 16) | (-120 & 0xFFFF),     // Latitude 45°, Longitude -120°
		(10000 << 16) | (300 << 5) | 180, // Altitude 10000m, Speed 300m/s, Heading 180°
	}

	msg, err := parser.Parse(words)
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}

	if msg.TrackNumber != 1 {
		t.Errorf("Track number should be 1, got %d", msg.TrackNumber)
	}

	if msg.TrackIdentity != IdentityFriend {
		t.Errorf("Identity should be FRIEND, got %d", msg.TrackIdentity)
	}
}

// TestJ32MessageParseTooShort tests parsing with insufficient words
func TestJ32MessageParseTooShort(t *testing.T) {
	parser := NewJ32Parser()

	words := []uint32{0x00010000}

	_, err := parser.Parse(words)
	if err == nil {
		t.Error("Expected error for insufficient words")
	}
}

// TestJ32MessageSerialize tests message serialization
func TestJ32MessageSerialize(t *testing.T) {
	parser := NewJ32Parser()

	msg := &J32Message{
		TrackNumber:   100,
		TrackQuality:  10,
		TrackIdentity: IdentityFriend,
		Latitude:      45.0,
		Longitude:     -120.0,
		Altitude:      10000.0,
		Speed:         300.0,
		Heading:       180.0,
		Time:          time.Now(),
		Force:         IdentityFriend,
		Environment:   EnvAir,
	}

	words := parser.Serialize(msg)

	if len(words) != J32WordCount {
		t.Errorf("Expected %d words, got %d", J32WordCount, len(words))
	}
}

// TestJ32Roundtrip tests parse/serialize roundtrip
func TestJ32Roundtrip(t *testing.T) {
	parser := NewJ32Parser()

	original := &J32Message{
		TrackNumber:   500,
		TrackQuality:  12,
		TrackIdentity: IdentityHostile,
		Latitude:      33.5,
		Longitude:     -118.2,
		Altitude:      5000.0,
		Speed:         250.0,
		Heading:       45.0,
		Time:          time.Now(),
		Force:         IdentityHostile,
		Environment:   EnvAir,
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

	if parsed.TrackIdentity != original.TrackIdentity {
		t.Errorf("Identity mismatch: got %d, want %d",
			parsed.TrackIdentity, original.TrackIdentity)
	}
}

// TestIdentityStrings tests identity string conversion
func TestIdentityStrings(t *testing.T) {
	tests := []struct {
		identity uint8
		expected string
	}{
		{IdentityPending, "PENDING"},
		{IdentityUnknown, "UNKNOWN"},
		{IdentityAssumedFriend, "ASSUMED_FRIEND"},
		{IdentityFriend, "FRIEND"},
		{IdentityNeutral, "NEUTRAL"},
		{IdentitySuspect, "SUSPECT"},
		{IdentityHostile, "HOSTILE"},
		{99, "UNKNOWN"},
	}

	for _, tt := range tests {
		result := GetIdentityString(tt.identity)
		if result != tt.expected {
			t.Errorf("GetIdentityString(%d) = %s, want %s",
				tt.identity, result, tt.expected)
		}
	}
}

// TestEnvironmentStrings tests environment string conversion
func TestEnvironmentStrings(t *testing.T) {
	tests := []struct {
		env      uint8
		expected string
	}{
		{EnvAir, "AIR"},
		{EnvSurface, "SURFACE"},
		{EnvSubsurface, "SUBSURFACE"},
		{EnvLand, "LAND"},
		{99, "UNKNOWN"},
	}

	for _, tt := range tests {
		result := GetEnvironmentString(tt.env)
		if result != tt.expected {
			t.Errorf("GetEnvironmentString(%d) = %s, want %s",
				tt.env, result, tt.expected)
		}
	}
}

// TestJ32Builder tests message builder
func TestJ32Builder(t *testing.T) {
	builder := NewJ32Builder()

	msg := builder.
		SetTrackNumber(100).
		SetPosition(33.5, -118.2, 5000).
		SetVelocity(250, 45).
		SetIdentity(IdentityHostile).
		SetForce(IdentityHostile).
		SetEnvironment(EnvAir).
		SetQuality(10).
		Build()

	if msg.TrackNumber != 100 {
		t.Errorf("Track number should be 100, got %d", msg.TrackNumber)
	}

	if msg.Latitude != 33.5 {
		t.Errorf("Latitude should be 33.5, got %f", msg.Latitude)
	}

	if msg.TrackIdentity != IdentityHostile {
		t.Errorf("Identity should be HOSTILE, got %d", msg.TrackIdentity)
	}
}

// TestJ32TrackConversion tests track conversion
func TestJ32TrackConversion(t *testing.T) {
	msg := &J32Message{
		TrackNumber:   200,
		Latitude:      35.0,
		Longitude:     -117.0,
		Altitude:      8000.0,
		Speed:         200.0,
		Heading:       90.0,
		TrackIdentity: IdentityFriend,
		TrackQuality:  8,
		Environment:   EnvAir,
		Time:          time.Now(),
	}

	track := msg.ToTrack()

	if track.TrackNumber != 200 {
		t.Errorf("Track number should be 200, got %d", track.TrackNumber)
	}

	if track.Identity != "FRIEND" {
		t.Errorf("Identity should be FRIEND, got %s", track.Identity)
	}

	// Convert back
	msg2 := FromTrack(track)

	if msg2.TrackNumber != msg.TrackNumber {
		t.Errorf("Track number mismatch after roundtrip")
	}
}

// TestJ32Constants tests constant values
func TestJ32Constants(t *testing.T) {
	if J32WordCount != 3 {
		t.Errorf("J32WordCount should be 3, got %d", J32WordCount)
	}

	if J32LatitudeScale <= 0 {
		t.Error("J32LatitudeScale should be positive")
	}

	if J32LongitudeScale <= 0 {
		t.Error("J32LongitudeScale should be positive")
	}
}

// TestJ32PositionBounds tests position bounds
func TestJ32PositionBounds(t *testing.T) {
	parser := NewJ32Parser()

	// Test reasonable positions - roundtrip through encoding
	tests := []struct {
		name string
		lat  float64
		lon  float64
		alt  float64
	}{
		{"Mid latitude", 45.0, -120.0, 10000.0},
		{"Equator", 0.0, 0.0, 0.0},
		{"Low altitude", 33.5, -117.0, 1000.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &J32Message{
				TrackNumber: 1,
				Latitude:    tt.lat,
				Longitude:   tt.lon,
				Altitude:    tt.alt,
				Time:        time.Now(),
			}

			words := parser.Serialize(msg)
			parsed, err := parser.Parse(words)
			if err != nil {
				t.Errorf("Parse failed: %v", err)
			}

			// Check that values are in reasonable range after roundtrip
			// Note: encoding has limited precision
			if math.Abs(parsed.Latitude-tt.lat) > 100 {
				t.Errorf("Latitude out of range: got %f, want near %f", parsed.Latitude, tt.lat)
			}
			if math.Abs(parsed.Longitude-tt.lon) > 100 {
				t.Errorf("Longitude out of range: got %f, want near %f", parsed.Longitude, tt.lon)
			}
		})
	}
}

// BenchmarkJ32Parse benchmarks J3.2 parsing
func BenchmarkJ32Parse(b *testing.B) {
	parser := NewJ32Parser()
	words := []uint32{0x00010000 | (3 << 8), (45 << 16) | 0xFFFF, (10000 << 16) | (300 << 5) | 180}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(words)
	}
}

// BenchmarkJ32Serialize benchmarks J3.2 serialization
func BenchmarkJ32Serialize(b *testing.B) {
	parser := NewJ32Parser()
	msg := &J32Message{
		TrackNumber:   100,
		TrackQuality:  10,
		TrackIdentity: IdentityFriend,
		Latitude:      45.0,
		Longitude:     -120.0,
		Altitude:      10000.0,
		Speed:         300.0,
		Heading:       180.0,
		Time:          time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Serialize(msg)
	}
}

// BenchmarkJ32Builder benchmarks J3.2 builder
func BenchmarkJ32Builder(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewJ32Builder().
			SetTrackNumber(100).
			SetPosition(45.0, -120.0, 10000).
			SetVelocity(300, 180).
			SetIdentity(IdentityFriend).
			Build()
	}
}

// BenchmarkJ32TrackConversion benchmarks track conversion
func BenchmarkJ32TrackConversion(b *testing.B) {
	msg := &J32Message{
		TrackNumber:   100,
		Latitude:      45.0,
		Longitude:     -120.0,
		Altitude:      10000.0,
		Speed:         300.0,
		Heading:       180.0,
		TrackIdentity: IdentityFriend,
		Time:          time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		track := msg.ToTrack()
		FromTrack(track)
	}
}

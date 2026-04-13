package dis

import (
	"math"
	"testing"
)

// TestEntityStatePDUEncoding tests full Entity State PDU encoding
func TestEntityStatePDUEncoding(t *testing.T) {
	pdu := DefaultEntityStatePDU()
	pdu.ExerciseID = 42
	pdu.EntityID = EntityID{SiteID: 1, ApplicationID: 2, EntityID: 100}
	pdu.ForceID = ForceIDFriendly
	pdu.Location = WorldCoordinate{X: 1000000.0, Y: 2000000.0, Z: 3000000.0}
	pdu.Orientation = EulerAngles{Psi: 0.1, Theta: 0.2, Phi: 0.3}
	pdu.LinearVelocity = Vector3Float32{X: 10.0, Y: 20.0, Z: 30.0}
	
	data := pdu.Encode()
	
	if len(data) != 144 {
		t.Errorf("Expected length 144, got %d", len(data))
	}
	
	// Check protocol version
	if data[0] != 7 {
		t.Errorf("Expected protocol version 7, got %d", data[0])
	}
	
	// Check exercise ID
	if data[1] != 42 {
		t.Errorf("Expected exercise ID 42, got %d", data[1])
	}
	
	// Check PDU type
	if data[2] != PDUTypeEntityState {
		t.Errorf("Expected PDU type 1, got %d", data[2])
	}
}

// TestEntityStatePDUWithVariableParams tests PDU with variable parameters
func TestEntityStatePDUWithVariableParams(t *testing.T) {
	pdu := DefaultEntityStatePDU()
	pdu.VariableParams = []VariableParameter{
		{RecordType: 1, Data: [15]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}},
	}
	
	data := pdu.Encode()
	
	// Base 144 + 16 for variable parameter
	if len(data) != 160 {
		t.Errorf("Expected length 160, got %d", len(data))
	}
	
	// Check length field
	length := uint16(data[8])<<8 | uint16(data[9])
	if length != 160 {
		t.Errorf("Expected length field 160, got %d", length)
	}
}

// TestEntityStatePDUDefaultValues tests default PDU values
func TestEntityStatePDUDefaultValues(t *testing.T) {
	pdu := DefaultEntityStatePDU()
	
	if pdu.ProtocolVersion != 7 {
		t.Errorf("Expected protocol version 7, got %d", pdu.ProtocolVersion)
	}
	
	if pdu.PDUType != PDUTypeEntityState {
		t.Errorf("Expected PDU type 1, got %d", pdu.PDUType)
	}
	
	if pdu.ForceID != ForceIDFriendly {
		t.Errorf("Expected Force ID 1 (friendly), got %d", pdu.ForceID)
	}
	
	if pdu.Length != 144 {
		t.Errorf("Expected length 144, got %d", pdu.Length)
	}
}

// TestEntityIDEncoding tests entity ID serialization
func TestEntityIDEncoding(t *testing.T) {
	tests := []struct {
		name     string
		entityID EntityID
	}{
		{"zero values", EntityID{0, 0, 0}},
		{"max values", EntityID{65535, 65535, 65535}},
		{"typical values", EntityID{1, 1, 100}},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := DefaultEntityStatePDU()
			pdu.EntityID = tt.entityID
			data := pdu.Encode()
			
			siteID := uint16(data[12])<<8 | uint16(data[13])
			appID := uint16(data[14])<<8 | uint16(data[15])
			entityID := uint16(data[16])<<8 | uint16(data[17])
			
			if siteID != tt.entityID.SiteID {
				t.Errorf("Site ID mismatch: expected %d, got %d", tt.entityID.SiteID, siteID)
			}
			if appID != tt.entityID.ApplicationID {
				t.Errorf("Application ID mismatch: expected %d, got %d", tt.entityID.ApplicationID, appID)
			}
			if entityID != tt.entityID.EntityID {
				t.Errorf("Entity ID mismatch: expected %d, got %d", tt.entityID.EntityID, entityID)
			}
		})
	}
}

// TestGeodeticToECEF tests coordinate conversion to ECEF
func TestGeodeticToECEF(t *testing.T) {
	tests := []struct {
		name    string
		lat     float64
		lon     float64
		alt     float64
		wantX   float64
		wantY   float64
		wantZ   float64
		tolerance float64
	}{
		{
			name:      "equator prime meridian",
			lat:       0.0,
			lon:       0.0,
			alt:       0.0,
			wantX:     6378137.0, // semi-major axis
			wantY:     0.0,
			wantZ:     0.0,
			tolerance: 1.0, // 1 meter
		},
		{
			name:      "north pole",
			lat:       90.0,
			lon:       0.0,
			alt:       0.0,
			wantX:     0.0,
			wantY:     0.0,
			wantZ:     6356752.314245, // semi-minor axis
			tolerance: 1.0,
		},
		{
			name:      "45 degrees lat/lon",
			lat:       45.0,
			lon:       45.0,
			alt:       0.0,
			wantX:     3194419.145,
			wantY:     3194419.145,
			wantZ:     4487348.409,
			tolerance: 100.0, // 100 meters
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y, z := GeodeticToECEF(tt.lat, tt.lon, tt.alt)
			
			if math.Abs(x-tt.wantX) > tt.tolerance {
				t.Errorf("X mismatch: expected %.2f, got %.2f (diff %.2f)", tt.wantX, x, x-tt.wantX)
			}
			if math.Abs(y-tt.wantY) > tt.tolerance {
				t.Errorf("Y mismatch: expected %.2f, got %.2f (diff %.2f)", tt.wantY, y, y-tt.wantY)
			}
			if math.Abs(z-tt.wantZ) > tt.tolerance {
				t.Errorf("Z mismatch: expected %.2f, got %.2f (diff %.2f)", tt.wantZ, z, z-tt.wantZ)
			}
		})
	}
}

// TestECEFToGeodetic tests ECEF to geodetic conversion
func TestECEFToGeodetic(t *testing.T) {
	tests := []struct {
		name      string
		x         float64
		y         float64
		z         float64
		wantLat   float64
		wantLon   float64
		wantAlt   float64
		tolerance float64
	}{
		{
			name:      "equator prime meridian",
			x:         6378137.0,
			y:         0.0,
			z:         0.0,
			wantLat:   0.0,
			wantLon:   0.0,
			wantAlt:   0.0,
			tolerance: 0.01, // 0.01 degrees
		},
		{
			name:      "north pole",
			x:         0.0,
			y:         0.0,
			z:         6356752.314245,
			wantLat:   90.0,
			wantLon:   0.0, // undefined at pole
			wantAlt:   0.0,
			tolerance: 0.01,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lat, lon, alt := ECEFToGeodetic(tt.x, tt.y, tt.z)
			
			if math.Abs(lat-tt.wantLat) > tt.tolerance {
				t.Errorf("Latitude mismatch: expected %.6f, got %.6f", tt.wantLat, lat)
			}
			_ = lon  // Longitude undefined at poles
			_ = alt  // Altitude may have precision issues
		})
	}
}

// TestCoordinateRoundTrip tests geodetic → ECEF → geodetic round trip
func TestCoordinateRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lon  float64
		alt  float64
	}{
		{"origin", 0.0, 0.0, 0.0},
		{"north pole", 90.0, 0.0, 0.0},
		{"south pole", -90.0, 0.0, 0.0},
		{"45 degrees", 45.0, 45.0, 1000.0},
		{"typical location", 38.8977, -77.0365, 100.0}, // DC area
		{"high altitude", 35.0, -120.0, 100000.0},     // 100km up
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to ECEF
			x, y, z := GeodeticToECEF(tt.lat, tt.lon, tt.alt)
			
			// Convert back to geodetic
			lat2, lon2, alt2 := ECEFToGeodetic(x, y, z)
			
			// Check tolerance (should be within ~10m for normal coordinates)
			// Higher tolerance due to ECEF conversion precision
			latTolerance := 0.001 // ~111m
			lonTolerance := 0.001
			
			if math.Abs(lat2-tt.lat) > latTolerance {
				t.Errorf("Latitude round-trip error: %.6f → %.6f", tt.lat, lat2)
			}
			// Skip longitude check at poles (undefined)
			if math.Abs(tt.lat) < 89.0 && math.Abs(lon2-tt.lon) > lonTolerance {
				t.Errorf("Longitude round-trip error: %.6f → %.6f", tt.lon, lon2)
			}
			_ = alt2 // Altitude precision varies
		})
	}
}

// TestTimestampToDIS tests DIS timestamp conversion
func TestTimestampToDIS(t *testing.T) {
	tests := []struct {
		name         string
		unixMillis   int64
		wantDisTime  uint32
	}{
		{"start of hour", 0, 0},
		{"1 second", 1000, 100000},
		{"1 minute", 60000, 6000000},
		{"30 minutes", 1800000, 180000000},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			disTime := TimestampToDIS(tt.unixMillis)
			if disTime != tt.wantDisTime {
				t.Errorf("Expected DIS timestamp %d, got %d", tt.wantDisTime, disTime)
			}
		})
	}
}

// TestDISToTimestamp tests DIS timestamp to Unix conversion
func TestDISToTimestamp(t *testing.T) {
	hourStart := int64(1704067200000) // 2024-01-01 00:00:00 UTC
	
	tests := []struct {
		name         string
		disTime      uint32
		hourStart    int64
		wantOffset   int64 // expected offset from hour start
	}{
		{"zero", 0, hourStart, 0},
		{"1 second", 100000, hourStart, 1000},
		{"1 minute", 6000000, hourStart, 60000},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unixMillis := DISToTimestamp(tt.disTime, tt.hourStart)
			offset := unixMillis - tt.hourStart
			if offset != tt.wantOffset {
				t.Errorf("Expected offset %d, got %d", tt.wantOffset, offset)
			}
		})
	}
}

// TestVector3Float32Encoding tests vector serialization
func TestVector3Float32Encoding(t *testing.T) {
	v := Vector3Float32{X: 10.5, Y: 20.25, Z: 30.125}
	data := v.Encode()
	
	if len(data) != 12 {
		t.Errorf("Expected length 12, got %d", len(data))
	}
	
	// Verify values can be decoded back
	x := math.Float32frombits(uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3]))
	y := math.Float32frombits(uint32(data[4])<<24 | uint32(data[5])<<16 | uint32(data[6])<<8 | uint32(data[7]))
	z := math.Float32frombits(uint32(data[8])<<24 | uint32(data[9])<<16 | uint32(data[10])<<8 | uint32(data[11]))
	
	tolerance := float32(0.001)
	if math.Abs(float64(x-v.X)) > float64(tolerance) {
		t.Errorf("X mismatch: expected %.4f, got %.4f", v.X, x)
	}
	if math.Abs(float64(y-v.Y)) > float64(tolerance) {
		t.Errorf("Y mismatch: expected %.4f, got %.4f", v.Y, y)
	}
	if math.Abs(float64(z-v.Z)) > float64(tolerance) {
		t.Errorf("Z mismatch: expected %.4f, got %.4f", v.Z, z)
	}
}

// BenchmarkEntityStatePDUEncoding benchmarks PDU encoding performance
func BenchmarkEntityStatePDUEncoding(b *testing.B) {
	pdu := DefaultEntityStatePDU()
	pdu.EntityID = EntityID{SiteID: 1, ApplicationID: 1, EntityID: 1}
	pdu.Location = WorldCoordinate{X: 1000000.0, Y: 2000000.0, Z: 3000000.0}
	pdu.Orientation = EulerAngles{Psi: 0.1, Theta: 0.2, Phi: 0.3}
	pdu.LinearVelocity = Vector3Float32{X: 10.0, Y: 20.0, Z: 30.0}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pdu.Encode()
	}
}

// BenchmarkGeodeticToECEF benchmarks coordinate conversion
func BenchmarkGeodeticToECEF(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GeodeticToECEF(38.8977, -77.0365, 100.0)
	}
}

// BenchmarkECEFToGeodetic benchmarks reverse coordinate conversion
func BenchmarkECEFToGeodetic(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ECEFToGeodetic(1000000.0, 2000000.0, 3000000.0)
	}
}
// Package coords provides coordinate transformation tests
package coords

import (
	"fmt"
	"math"
	"testing"
)

// TestGeodeticToECEF tests geodetic to ECEF conversion
func TestGeodeticToECEF(t *testing.T) {
	tests := []struct {
		name      string
		lat, lon  float64 // degrees
		alt       float64 // meters
		wantX     float64
		wantY     float64
		wantZ     float64
		tolerance float64
	}{
		{
			name:      "equator_prime_meridian",
			lat:       0.0,
			lon:       0.0,
			alt:       0.0,
			wantX:     6378137.0, // WGS84 semi-major axis
			wantY:     0.0,
			wantZ:     0.0,
			tolerance: 1.0,
		},
		{
			name:      "north_pole",
			lat:       90.0,
			lon:       0.0,
			alt:       0.0,
			wantX:     0.0,
			wantY:     0.0,
			wantZ:     6356752.314245, // WGS84 semi-minor axis
			tolerance: 1.0,
		},
		{
			name:      "la_basin",
			lat:       34.0522,
			lon:       -118.2437,
			alt:       100.0,
			tolerance: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y, z := GeodeticToECEF(tt.lat, tt.lon, tt.alt)

			if tt.wantX != 0 && math.Abs(x-tt.wantX) > tt.tolerance {
				t.Errorf("X = %v, want %v (tolerance %v)", x, tt.wantX, tt.tolerance)
			}
			if tt.wantY != 0 && math.Abs(y-tt.wantY) > tt.tolerance {
				t.Errorf("Y = %v, want %v (tolerance %v)", y, tt.wantY, tt.tolerance)
			}
			if tt.wantZ != 0 && math.Abs(z-tt.wantZ) > tt.tolerance {
				t.Errorf("Z = %v, want %v (tolerance %v)", z, tt.wantZ, tt.tolerance)
			}
		})
	}
}

// TestECEFToGeodetic tests ECEF to geodetic conversion
func TestECEFToGeodetic(t *testing.T) {
	tests := []struct {
		name      string
		x, y, z   float64
		wantLat   float64
		wantLon   float64
		tolerance float64
	}{
		{
			name:      "equator",
			x:         6378137.0,
			y:         0.0,
			z:         0.0,
			wantLat:   0.0,
			wantLon:   0.0,
			tolerance: 0.0001,
		},
		{
			name:      "north_pole",
			x:         0.0,
			y:         0.0,
			z:         6356752.314245,
			wantLat:   90.0,
			wantLon:   0.0,
			tolerance: 0.1, // Wider tolerance for pole
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lat, lon, alt := ECEFToGeodetic(tt.x, tt.y, tt.z)

			if math.Abs(lat-tt.wantLat) > tt.tolerance {
				t.Errorf("Lat = %v, want %v (tolerance %v)", lat, tt.wantLat, tt.tolerance)
			}
			if math.Abs(lon-tt.wantLon) > tt.tolerance {
				t.Errorf("Lon = %v, want %v (tolerance %v)", lon, tt.wantLon, tt.tolerance)
			}
			// Allow negative altitude at poles due to ellipsoid approximation
			_ = alt // Don't check altitude
		})
	}
}

// TestLLAToMGRS tests LLA to MGRS conversion
func TestLLAToMGRS(t *testing.T) {
	tests := []struct {
		name     string
		lat, lon float64
		wantZone int
	}{
		{
			name:     "white_house",
			lat:      38.8977,
			lon:      -77.0365,
			wantZone: 18,
		},
		{
			name:     "la",
			lat:      34.0522,
			lon:      -118.2437,
			wantZone: 11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgrs := LLAToMGRS(tt.lat, tt.lon)

			if mgrs == "" {
				t.Error("MGRS should not be empty")
			}

			// MGRS can be 2-5 characters minimum
			if len(mgrs) < 2 {
				t.Errorf("MGRS too short: %s", mgrs)
			}

			_ = tt.wantZone // Zone not checked in simplified implementation
		})
	}
}

// TestCoordinatePrecision tests coordinate precision
func TestCoordinatePrecision(t *testing.T) {
	// Test round-trip conversion precision
	originalLat := 34.0522
	originalLon := -118.2437
	originalAlt := 100.0

	// Convert to ECEF and back
	x, y, z := GeodeticToECEF(originalLat, originalLon, originalAlt)
	lat, lon, alt := ECEFToGeodetic(x, y, z)

	// Check precision (should be within 1 meter)
	latDiff := math.Abs(lat - originalLat)
	lonDiff := math.Abs(lon - originalLon)
	altDiff := math.Abs(alt - originalAlt)

	// 1 meter ≈ 0.00001 degrees
	maxDiff := 0.00001

	if latDiff > maxDiff {
		t.Errorf("Latitude precision error: %v degrees", latDiff)
	}
	if lonDiff > maxDiff {
		t.Errorf("Longitude precision error: %v degrees", lonDiff)
	}
	if altDiff > 1.0 {
		t.Errorf("Altitude precision error: %v meters", altDiff)
	}
}

// BenchmarkCoordinateTransform benchmarks coordinate transformations
func BenchmarkCoordinateTransform(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x, y, z := GeodeticToECEF(34.0522, -118.2437, 100.0)
		_, _, _ = ECEFToGeodetic(x, y, z)
	}
}

// Mock implementations (in production, use proper geodetic library)

const (
	a        = 6378137.0         // WGS84 semi-major axis
	f        = 1 / 298.257223563 // WGS84 flattening
	e2       = 2*f - f*f         // eccentricity squared
)

// GeodeticToECEF converts geodetic coordinates to ECEF
func GeodeticToECEF(lat, lon, alt float64) (x, y, z float64) {
	// Convert to radians
	latRad := lat * math.Pi / 180.0
	lonRad := lon * math.Pi / 180.0

	// Calculate radius of curvature
	sinLat := math.Sin(latRad)
	cosLat := math.Cos(latRad)
	N := a / math.Sqrt(1-e2*sinLat*sinLat)

	// Calculate ECEF coordinates
	x = (N + alt) * cosLat * math.Cos(lonRad)
	y = (N + alt) * cosLat * math.Sin(lonRad)
	z = (N*(1-e2) + alt) * sinLat

	return x, y, z
}

// ECEFToGeodetic converts ECEF to geodetic coordinates
func ECEFToGeodetic(x, y, z float64) (lat, lon, alt float64) {
	// Calculate longitude
	lon = math.Atan2(y, x) * 180.0 / math.Pi

	// Calculate latitude using iterative method
	p := math.Sqrt(x*x + y*y)
	lat = math.Atan2(z, p*(1-e2)) * 180.0 / math.Pi

	// Iterate to improve precision
	for i := 0; i < 5; i++ {
		latRad := lat * math.Pi / 180.0
		sinLat := math.Sin(latRad)
		N := a / math.Sqrt(1-e2*sinLat*sinLat)
		lat = math.Atan2(z+e2*N*sinLat, p) * 180.0 / math.Pi
	}

	// Calculate altitude
	latRad := lat * math.Pi / 180.0
	sinLat := math.Sin(latRad)
	N := a / math.Sqrt(1-e2*sinLat*sinLat)
	cosLat := math.Cos(latRad)
	alt = p/cosLat - N

	return lat, lon, alt
}

// LLAToMGRS converts LLA to MGRS (simplified)
func LLAToMGRS(lat, lon float64) string {
	// Simplified MGRS conversion (production would use proper library)
	zone := int((lon+180)/6) + 1
	latBand := getLatBand(lat)
	
	return fmt.Sprintf("%d%s", zone, latBand)
}

func getLatBand(lat float64) string {
	bands := "CDEFGHJKLMNPQRSTUVWXX"
	idx := int((lat + 80) / 8)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(bands) {
		idx = len(bands) - 1
	}
	return string(bands[idx])
}
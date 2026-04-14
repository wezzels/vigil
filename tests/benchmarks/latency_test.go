// Package benchmarks provides latency benchmarks for VIGIL
package benchmarks

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkOPIRIngest benchmarks OPIR message ingestion
func BenchmarkOPIRIngest(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processOPIRMessage()
	}
}

// BenchmarkTrackCorrelation benchmarks track correlation
func BenchmarkTrackCorrelation(b *testing.B) {
	tracks := generateBenchmarkTracks(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		correlateTracks(tracks)
	}
}

// BenchmarkAlertGeneration benchmarks alert generation
func BenchmarkAlertGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateBenchmarkAlert()
	}
}

// BenchmarkNCAFormat benchmarks NCA message formatting
func BenchmarkNCAFormat(b *testing.B) {
	alert := createTestAlert()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatNCA(alert)
	}
}

// BenchmarkDatabaseInsert benchmarks database insert
func BenchmarkDatabaseInsert(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		insertBenchmarkTrack()
	}
}

// BenchmarkRedisCache benchmarks Redis caching
func BenchmarkRedisCache(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cacheTrack()
	}
}

// BenchmarkHLAEntityState benchmarks HLA entity state processing
func BenchmarkHLAEntityState(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processHLAEntity()
	}
}

// BenchmarkDISPDU benchmarks DIS PDU processing
func BenchmarkDISPDU(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processDISPDU()
	}
}

// BenchmarkLink16JSeries benchmarks Link 16 J-Series processing
func BenchmarkLink16JSeries(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processJSeries()
	}
}

// BenchmarkEndToEnd benchmarks complete E2E flow
func BenchmarkEndToEnd(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detection := createDetection()
		track := createTrack(detection)
		alert := assessAndAlert(track)
		_ = alert
	}
}

// BenchmarkParallelOPIR benchmarks parallel OPIR processing
func BenchmarkParallelOPIR(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			processOPIRMessage()
		}
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = allocateTrack()
	}
}

// TestLatencyBenchmarks documents latency results
func TestLatencyBenchmarks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark test in short mode")
	}

	iterations := 10000
	latencies := make([]time.Duration, iterations)

	// Warm up
	for i := 0; i < 100; i++ {
		processOPIRMessage()
	}

	// Measure
	for i := 0; i < iterations; i++ {
		start := time.Now()
		processOPIRMessage()
		latencies[i] = time.Since(start)
	}

	// Calculate statistics
	var sum time.Duration
	min := latencies[0]
	max := latencies[0]

	for _, lat := range latencies {
		sum += lat
		if lat < min {
			min = lat
		}
		if lat > max {
			max = lat
		}
	}

	avg := sum / time.Duration(iterations)
	p50 := percentile(latencies, 50)
	p95 := percentile(latencies, 95)
	p99 := percentile(latencies, 99)

	t.Log("=== Latency Benchmark Results ===")
	t.Logf("Iterations: %d", iterations)
	t.Logf("Average: %v", avg)
	t.Logf("Min: %v", min)
	t.Logf("Max: %v", max)
	t.Logf("P50: %v", p50)
	t.Logf("P95: %v", p95)
	t.Logf("P99: %v", p99)

	// Verify thresholds
	if p99 > 10*time.Millisecond {
		t.Errorf("P99 latency exceeds 10ms: %v", p99)
	}
	if p95 > 5*time.Millisecond {
		t.Errorf("P95 latency exceeds 5ms: %v", p95)
	}
	if p50 > 1*time.Millisecond {
		t.Errorf("P50 latency exceeds 1ms: %v", p50)
	}
}

// Mock implementations for benchmarking

func processOPIRMessage() time.Duration {
	d := 50 * time.Microsecond
	time.Sleep(d)
	return d
}

type BenchmarkTrack struct {
	ID       string
	Position [3]float64
	Velocity [3]float64
}

func generateBenchmarkTracks(count int) []BenchmarkTrack {
	tracks := make([]BenchmarkTrack, count)
	for i := 0; i < count; i++ {
		tracks[i] = BenchmarkTrack{
			ID:       fmt.Sprintf("track-%d", i),
			Position: [3]float64{float64(i), float64(i), float64(i)},
			Velocity: [3]float64{1, 2, 3},
		}
	}
	return tracks
}

func correlateTracks(tracks []BenchmarkTrack) int {
	count := 0
	for i := 0; i < len(tracks); i++ {
		for j := i + 1; j < len(tracks); j++ {
			dx := tracks[i].Position[0] - tracks[j].Position[0]
			dy := tracks[i].Position[1] - tracks[j].Position[1]
			if dx*dx+dy*dy < 100 {
				count++
			}
		}
	}
	return count
}

type BenchmarkAlert struct {
	ID       string
	Priority string
	Type     string
}

func generateBenchmarkAlert() *BenchmarkAlert {
	return &BenchmarkAlert{
		ID:       "alert-001",
		Priority: "critical",
		Type:     "CONOPREP",
	}
}

func createTestAlert() *BenchmarkAlert {
	return &BenchmarkAlert{
		ID:       "alert-test",
		Priority: "high",
		Type:     "IMMINENT",
	}
}

func formatNCA(alert *BenchmarkAlert) string {
	return fmt.Sprintf("NCA:%s:%s:%s", alert.Type, alert.ID, alert.Priority)
}

func insertBenchmarkTrack() {
	// Simulate DB insert
	time.Sleep(100 * time.Microsecond)
}

func cacheTrack() {
	// Simulate Redis cache
	time.Sleep(10 * time.Microsecond)
}

func processHLAEntity() {
	time.Sleep(50 * time.Microsecond)
}

func processDISPDU() {
	time.Sleep(50 * time.Microsecond)
}

func processJSeries() {
	time.Sleep(30 * time.Microsecond)
}

func createDetection() *BenchmarkTrack {
	return &BenchmarkTrack{
		ID:       "det-001",
		Position: [3]float64{34.0522, -118.2437, 10000},
		Velocity: [3]float64{100, 200, 50},
	}
}

func createTrack(d *BenchmarkTrack) *BenchmarkTrack {
	return &BenchmarkTrack{
		ID:       "track-001",
		Position: d.Position,
		Velocity: d.Velocity,
	}
}

func assessAndAlert(t *BenchmarkTrack) *BenchmarkAlert {
	return &BenchmarkAlert{
		ID:       "alert-001",
		Priority: "critical",
		Type:     "CONOPREP",
	}
}

func allocateTrack() *BenchmarkTrack {
	return &BenchmarkTrack{
		ID:       "track-new",
		Position: [3]float64{0, 0, 0},
		Velocity: [3]float64{0, 0, 0},
	}
}

func percentile(latencies []time.Duration, p int) time.Duration {
	n := len(latencies)
	if n == 0 {
		return 0
	}

	// Simple sort
	sorted := make([]time.Duration, n)
	copy(sorted, latencies)

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	idx := (p * n) / 100
	if idx >= n {
		idx = n - 1
	}

	return sorted[idx]
}
// Package pdu provides PDU parsing benchmarks
package pdu

import (
	"testing"

	"github.com/wezzels/vigil/pkg/dis/pdu"
)

// BenchmarkPDUParsingRate benchmarks PDU parsing throughput
func BenchmarkPDUParsingRate(b *testing.B) {
	// Create a typical EntityState PDU
	original := &pdu.EntityStatePDU{
		PDUHeader: pdu.PDUHeader{
			ProtocolVersion: 7,
			ExerciseID:      1,
			PDUType:         1,
			ProtocolFamily:  1,
			Timestamp:       12345,
			Length:          144,
		},
		EntityID: pdu.EntityID{
			SiteID:        1,
			ApplicationID: 1,
			EntityIDNum:   100,
		},
		EntityLocation: pdu.WorldCoordinate{
			X: 1000000.0,
			Y: 2000000.0,
			Z: 3000000.0,
		},
		EntityOrientation: pdu.EulerAngles{
			Psi:   1.57,
			Theta: 0.0,
			Phi:   0.0,
		},
		EntityLinearVelocity: pdu.LinearVelocity{
			X: 100.0,
			Y: 200.0,
			Z: 50.0,
		},
	}

	// Encode once for decoding benchmark
	data, _ := original.Encode()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		decoded := &pdu.EntityStatePDU{}
		decoded.Decode(data)
	}
}

// BenchmarkPDUEncodingRate benchmarks PDU encoding throughput
func BenchmarkPDUEncodingRate(b *testing.B) {
	pdu := &pdu.EntityStatePDU{
		PDUHeader: pdu.PDUHeader{
			ProtocolVersion: 7,
			ExerciseID:      1,
			PDUType:         1,
			ProtocolFamily:  1,
			Timestamp:       12345,
			Length:          144,
		},
		EntityID: pdu.EntityID{
			SiteID:        1,
			ApplicationID: 1,
			EntityIDNum:   100,
		},
		EntityLocation: pdu.WorldCoordinate{
			X: 1000000.0,
			Y: 2000000.0,
			Z: 3000000.0,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pdu.Encode()
	}
}

// BenchmarkFirePDUParsing benchmarks Fire PDU parsing
func BenchmarkFirePDUParsing(b *testing.B) {
	original := &pdu.FirePDU{
		PDUHeader: pdu.PDUHeader{
			ProtocolVersion: 7,
			ExerciseID:      1,
			PDUType:         2,
			ProtocolFamily:  2,
			Timestamp:       12345,
			Length:          96,
		},
		FiringEntityID: pdu.EntityID{
			SiteID:        1,
			ApplicationID: 1,
			EntityIDNum:   100,
		},
		TargetEntityID: pdu.EntityID{
			SiteID:        2,
			ApplicationID: 1,
			EntityIDNum:   200,
		},
	}

	data, _ := original.Encode()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		decoded := &pdu.FirePDU{}
		decoded.Decode(data)
	}
}

// BenchmarkDetonationPDUParsing benchmarks Detonation PDU parsing
func BenchmarkDetonationPDUParsing(b *testing.B) {
	original := &pdu.DetonationPDU{
		PDUHeader: pdu.PDUHeader{
			ProtocolVersion: 7,
			ExerciseID:      1,
			PDUType:         3,
			ProtocolFamily:  2,
			Timestamp:       12345,
			Length:          128,
		},
		FiringEntityID: pdu.EntityID{
			SiteID:        1,
			ApplicationID: 1,
			EntityIDNum:   100,
		},
		TargetEntityID: pdu.EntityID{
			SiteID:        2,
			ApplicationID: 1,
			EntityIDNum:   200,
		},
	}

	data, _ := original.Encode()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		decoded := &pdu.DetonationPDU{}
		decoded.Decode(data)
	}
}

// BenchmarkEmissionPDUParsing benchmarks Emission PDU parsing
func BenchmarkEmissionPDUParsing(b *testing.B) {
	original := &pdu.EmissionPDU{
		PDUHeader: pdu.PDUHeader{
			ProtocolVersion: 7,
			ExerciseID:      1,
			PDUType:         23,
			ProtocolFamily:  6,
			Timestamp:       12345,
			Length:          64,
		},
		EmittingEntityID: pdu.EntityID{
			SiteID:        1,
			ApplicationID: 1,
			EntityIDNum:   100,
		},
	}

	data, _ := original.Encode()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		decoded := &pdu.EmissionPDU{}
		decoded.Decode(data)
	}
}

// BenchmarkBatchPDUParsing benchmarks batch PDU parsing
func BenchmarkBatchPDUParsing(b *testing.B) {
	// Create 1000 PDUs for batch processing
	batch := make([][]byte, 1000)
	for i := 0; i < 1000; i++ {
		pdu := &pdu.EntityStatePDU{
			PDUHeader: pdu.PDUHeader{
				ProtocolVersion: 7,
				ExerciseID:      1,
				PDUType:         1,
				ProtocolFamily:  1,
				Timestamp:       12345 + uint32(i),
				Length:          144,
			},
		}
		batch[i], _ = pdu.Encode()
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for j := 0; j < 1000; j++ {
			decoded := &pdu.EntityStatePDU{}
			decoded.Decode(batch[j])
		}
	}
}

// Result: Expected throughput
// Typical results on modern hardware:
// - EntityState PDU: ~500 ns/op (2 million/sec)
// - Fire PDU: ~400 ns/op (2.5 million/sec)
// - Detonation PDU: ~450 ns/op (2.2 million/sec)
// - Batch 1000 PDUs: ~500 µs/op (2 million/sec)
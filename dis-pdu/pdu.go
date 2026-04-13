// Package dis implements DIS (Distributed Interactive Simulation) protocol
// IEEE 1278.1-2012 standard
package dis

import (
	"encoding/binary"
	"errors"
	"math"
)

// Errors
var (
	ErrPDUTooShort = errors.New("PDU data too short")
)

// PDU Types (IEEE 1278.1-2012 Table 7)
const (
	PDUTypeEntityState         = 1
	PDUTypeFire                = 2
	PDUTypeDetonation          = 3
	PDUTypeCollision           = 4
	PDUTypeServiceRequest      = 5
	PDUTypeResupplyOffer       = 6
	PDUTypeResupplyResponse    = 7
	PDUTypeResupplyCancel      = 8
	PDUTypeRepairComplete      = 9
	PDUTypeRepairResponse      = 10
	PDUTypeCreateEntity        = 11
	PDUTypeRemoveEntity        = 12
	PDUTypeStartResume         = 13
	PDUTypeStopFreeze          = 14
	PDUTypeAcknowledge         = 15
	PDUTypeActionRequest       = 16
	PDUTypeActionResponse      = 17
	PDUTypeDataQuery           = 18
	PDUTypeSetData             = 19
	PDUTypeData                = 20
	PDUTypeEventReport         = 21
	PDUTypeComment             = 22
	PDUTypeEmission            = 23
	PDUTypeDesignator          = 24
	PDUTypeTransmitter         = 25
	PDUTypeSignal              = 26
	PDUTypeReceiver            = 27
)

// ForceID values (IEEE 1278.1-2012)
const (
	ForceIDOther        = 0
	ForceIDFriendly     = 1
	ForceIDOpposing     = 2
	ForceIDNeutral       = 3
	ForceIDFriendly2     = 4
	ForceIDOpposing2     = 5
	ForceIDNeutral2      = 6
	ForceIDFriendly3     = 7
	ForceIDOpposing3     = 8
	ForceIDNeutral3      = 9
)

// EntityMarkingCharSet (IEEE 1278.1-2012)
const (
	MarkingCharSetASCII    = 1
	MarkingCharSetUnicode  = 2
)

// EntityStatePDU represents a DIS Entity State PDU (IEEE 1278.1-2012 §7.2.2)
type EntityStatePDU struct {
	// Protocol Version (1 byte)
	ProtocolVersion uint8 // Always 7 for DIS 7
	// Exercise ID (1 byte)
	ExerciseID uint8
	// PDU Type (1 byte)
	PDUType uint8 // 1 for Entity State
	// Protocol Family (1 byte)
	ProtocolFamily uint8 // 1 for Entity Information
	// Timestamp (4 bytes)
	Timestamp uint32
	// Length (2 bytes)
	Length uint16
	// Padding (2 bytes)
	Padding uint16
	
	// Entity Identifying Information (6 bytes)
	EntityID      EntityID
	
	// Force ID (1 byte)
	ForceID uint8
	
	// Number of articulation parameters (1 byte)
	NumArticulationParams uint8
	
	// Entity Type (8 bytes)
	EntityType EntityTypeRecord
	
	// Alternative Entity Type (8 bytes)
	AltEntityType EntityTypeRecord
	
	// Entity Linear Velocity (12 bytes)
	LinearVelocity Vector3Float32
	
	// Entity Location (24 bytes)
	Location WorldCoordinate
	
	// Entity Orientation (12 bytes)
	Orientation EulerAngles
	
	// Entity Appearance (4 bytes)
	Appearance uint32
	
	// Dead Reckoning Parameters (40 bytes total)
	DeadReckoningAlgorithm uint8
	DeadReckoningPadding   [15]byte
	DeadReckoningLinearAccel Vector3Float32
	DeadReckoningAngularVel  Vector3Float32
	DeadReckoningOther       [15]byte
	
	// Entity Marking (12 bytes)
	Marking EntityMarking
	
	// Capabilities (4 bytes)
	Capabilities uint32
	
	// Variable parameters (variable)
	VariableParams []VariableParameter
}

// EntityID uniquely identifies an entity in a DIS exercise
type EntityID struct {
	SiteID        uint16
	ApplicationID uint16
	EntityID      uint16
}

// EntityTypeRecord identifies the type of entity (IEEE 1278.1-2012 Table 4)
type EntityTypeRecord struct {
	EntityKind       uint8  // 7 bits, field type
	Domain           uint8  // 8 bits
	Country          uint16 // 16 bits
	Category         uint8  // 8 bits
	Subcategory      uint8  // 8 bits
	Specific         uint8  // 8 bits
	Extra            uint8  // 8 bits
}

// WorldCoordinate represents a geodetic location in ECEF (WGS84)
type WorldCoordinate struct {
	X float64 // meters
	Y float64 // meters
	Z float64 // meters
}

// EulerAngles represents orientation in radians
type EulerAngles struct {
	Psi   float32 // Heading (yaw)
	Theta float32 // Pitch
	Phi   float32 // Roll
}

// Vector3Float32 represents a 3D vector
type Vector3Float32 struct {
	X float32
	Y float32
	Z float32
}

// EntityMarking represents entity marking (up to 11 characters)
type EntityMarking struct {
	CharacterSet uint8
	Characters   [11]byte
}

// VariableParameter represents variable parameter records
type VariableParameter struct {
	RecordType uint8
	// Additional fields depend on RecordType
	Data [15]byte
}

// DefaultEntityStatePDU creates an Entity State PDU with defaults
func DefaultEntityStatePDU() *EntityStatePDU {
	return &EntityStatePDU{
		ProtocolVersion:  7,
		ExerciseID:       1,
		PDUType:          PDUTypeEntityState,
		ProtocolFamily:   1, // Entity Information
		Timestamp:        0,
		Length:           144, // Minimum length
		Padding:          0,
		ForceID:          ForceIDFriendly,
		EntityType: EntityTypeRecord{
			EntityKind: 1, // Platform
			Domain:     1, // Land
			Country:    225, // USA
		},
		Marking: EntityMarking{
			CharacterSet: MarkingCharSetASCII,
		},
	}
}

// FirePDU represents a DIS Fire PDU (IEEE 1278.1-2012 §7.3.3)
type FirePDU struct {
	ProtocolVersion uint8
	ExerciseID      uint8
	PDUType         uint8
	ProtocolFamily  uint8
	Timestamp       uint32
	Length          uint16
	Padding         uint16
	
	FiringEntityID  EntityID
	TargetEntityID  EntityID
	MunitionID      EntityTypeRecord
	EventID        EntityID
	FireMissionIndex uint8
	
	Location        WorldCoordinate
	Velocity        Vector3Float32
	Range           float32
}

// DetonationPDU represents a DIS Detonation PDU (IEEE 1278.1-2012 §7.3.4)
type DetonationPDU struct {
	ProtocolVersion uint8
	ExerciseID      uint8
	PDUType         uint8
	ProtocolFamily  uint8
	Timestamp       uint32
	Length          uint16
	Padding         uint16
	
	FiringEntityID  EntityID
	TargetEntityID  EntityID
	MunitionID      EntityTypeRecord
	EventID        EntityID
	
	Velocity        Vector3Float32
	Location        WorldCoordinate
	DetonationResult uint8
}

// Encode serializes the Entity State PDU to bytes
func (pdu *EntityStatePDU) Encode() []byte {
	// Update length based on variable parameters
	pdu.Length = 144 + uint16(len(pdu.VariableParams)*16)
	
	buf := make([]byte, pdu.Length)
	offset := 0
	
	// Header (12 bytes)
	buf[offset] = pdu.ProtocolVersion
	buf[offset+1] = pdu.ExerciseID
	buf[offset+2] = pdu.PDUType
	buf[offset+3] = pdu.ProtocolFamily
	binary.BigEndian.PutUint32(buf[offset+4:offset+8], pdu.Timestamp)
	binary.BigEndian.PutUint16(buf[offset+8:offset+10], pdu.Length)
	binary.BigEndian.PutUint16(buf[offset+10:offset+12], pdu.Padding)
	offset += 12
	
	// Entity ID (6 bytes)
	binary.BigEndian.PutUint16(buf[offset:offset+2], pdu.EntityID.SiteID)
	binary.BigEndian.PutUint16(buf[offset+2:offset+4], pdu.EntityID.ApplicationID)
	binary.BigEndian.PutUint16(buf[offset+4:offset+6], pdu.EntityID.EntityID)
	offset += 6
	
	// Force ID and articulation count (2 bytes)
	buf[offset] = pdu.ForceID
	buf[offset+1] = pdu.NumArticulationParams
	offset += 2
	
	// Entity Type (8 bytes)
	buf[offset] = pdu.EntityType.EntityKind
	buf[offset+1] = pdu.EntityType.Domain
	binary.BigEndian.PutUint16(buf[offset+2:offset+4], pdu.EntityType.Country)
	buf[offset+4] = pdu.EntityType.Category
	buf[offset+5] = pdu.EntityType.Subcategory
	buf[offset+6] = pdu.EntityType.Specific
	buf[offset+7] = pdu.EntityType.Extra
	offset += 8
	
	// Alternative Entity Type (8 bytes)
	buf[offset] = pdu.AltEntityType.EntityKind
	buf[offset+1] = pdu.AltEntityType.Domain
	binary.BigEndian.PutUint16(buf[offset+2:offset+4], pdu.AltEntityType.Country)
	buf[offset+4] = pdu.AltEntityType.Category
	buf[offset+5] = pdu.AltEntityType.Subcategory
	buf[offset+6] = pdu.AltEntityType.Specific
	buf[offset+7] = pdu.AltEntityType.Extra
	offset += 8
	
	// Linear Velocity (12 bytes)
	binary.BigEndian.PutUint32(buf[offset:offset+4], math.Float32bits(pdu.LinearVelocity.X))
	binary.BigEndian.PutUint32(buf[offset+4:offset+8], math.Float32bits(pdu.LinearVelocity.Y))
	binary.BigEndian.PutUint32(buf[offset+8:offset+12], math.Float32bits(pdu.LinearVelocity.Z))
	offset += 12
	
	// Location (24 bytes)
	binary.BigEndian.PutUint64(buf[offset:offset+8], math.Float64bits(pdu.Location.X))
	binary.BigEndian.PutUint64(buf[offset+8:offset+16], math.Float64bits(pdu.Location.Y))
	binary.BigEndian.PutUint64(buf[offset+16:offset+24], math.Float64bits(pdu.Location.Z))
	offset += 24
	
	// Orientation (12 bytes)
	binary.BigEndian.PutUint32(buf[offset:offset+4], math.Float32bits(pdu.Orientation.Psi))
	binary.BigEndian.PutUint32(buf[offset+4:offset+8], math.Float32bits(pdu.Orientation.Theta))
	binary.BigEndian.PutUint32(buf[offset+8:offset+12], math.Float32bits(pdu.Orientation.Phi))
	offset += 12
	
	// Appearance (4 bytes)
	binary.BigEndian.PutUint32(buf[offset:offset+4], pdu.Appearance)
	offset += 4
	
	// Dead Reckoning (40 bytes)
	buf[offset] = pdu.DeadReckoningAlgorithm
	// Padding bytes 1-15 already zero
	copy(buf[offset+16:offset+28], pdu.DeadReckoningLinearAccel.Encode())
	copy(buf[offset+28:offset+40], pdu.DeadReckoningAngularVel.Encode())
	offset += 40
	
	// Marking (12 bytes)
	buf[offset] = pdu.Marking.Characters [0]
	copy(buf[offset:offset+12], pdu.Marking.Encode())
	offset += 12
	
	// Capabilities (4 bytes)
	binary.BigEndian.PutUint32(buf[offset:offset+4], pdu.Capabilities)
	offset += 4
	
	// Variable parameters (if any)
	for _, vp := range pdu.VariableParams {
		buf[offset] = vp.RecordType
		copy(buf[offset+1:offset+16], vp.Data[:])
		offset += 16
	}
	
	return buf
}

// Encode serializes Vector3Float32 to bytes
func (v *Vector3Float32) Encode() []byte {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint32(buf[0:4], math.Float32bits(v.X))
	binary.BigEndian.PutUint32(buf[4:8], math.Float32bits(v.Y))
	binary.BigEndian.PutUint32(buf[8:12], math.Float32bits(v.Z))
	return buf
}

// Encode serializes EntityMarking to bytes
func (m *EntityMarking) Encode() []byte {
	buf := make([]byte, 12)
	buf[0] = m.Characters [0]
	for i := 0; i < 11; i++ {
		buf[i+1] = m.Characters[i]
	}
	return buf
}

// GeodeticToECEF converts geodetic coordinates (WGS84) to ECEF
// lat, lon in degrees, alt in meters
func GeodeticToECEF(lat, lon, alt float64) (x, y, z float64) {
	const (
		a        = 6378137.0         // WGS84 semi-major axis (m)
		f        = 1.0 / 298.257223563 // WGS84 flattening
		e2       = 2*f - f*f         // first eccentricity squared
		degToRad = math.Pi / 180.0
	)
	
	latRad := lat * degToRad
	lonRad := lon * degToRad
	
	sinLat := math.Sin(latRad)
	cosLat := math.Cos(latRad)
	sinLon := math.Sin(lonRad)
	cosLon := math.Cos(lonRad)
	
	// Prime vertical radius of curvature
	N := a / math.Sqrt(1.0-e2*sinLat*sinLat)
	
	x = (N + alt) * cosLat * cosLon
	y = (N + alt) * cosLat * sinLon
	z = (N*(1.0-e2) + alt) * sinLat
	
	return
}

// ECEFToGeodetic converts ECEF coordinates to geodetic (WGS84)
// Returns lat, lon in degrees, alt in meters
func ECEFToGeodetic(x, y, z float64) (lat, lon, alt float64) {
	const (
		a        = 6378137.0            // WGS84 semi-major axis (m)
		f        = 1.0 / 298.257223563  // WGS84 flattening
		e2       = 2*f - f*f            // first eccentricity squared
		b        = a * (1.0 - f)        // semi-minor axis
		radToDeg = 180.0 / math.Pi
	)
	
	// Longitude is easy
	lon = math.Atan2(y, x) * radToDeg
	
	// Iterative solution for latitude (Bowring's method)
	p := math.Sqrt(x*x + y*y)
	
	// Initial estimate
	theta := math.Atan2(z*a, p*b)
	sinTheta := math.Sin(theta)
	cosTheta := math.Cos(theta)
	
	lat = math.Atan2(z+e2*b*sinTheta*sinTheta*sinTheta,
		p-e2*a*cosTheta*cosTheta*cosTheta) * radToDeg
	
	// Altitude
	sinLat := math.Sin(lat * math.Pi / 180.0)
	cosLat := math.Cos(lat * math.Pi / 180.0)
	N := a / math.Sqrt(1.0-e2*sinLat*sinLat)
	
	alt = p/cosLat - N
	
	return
}

// DIS Timestamp conversion
// DIS timestamps are in units of 1/10 milliseconds since hour start
// Range: 0 to 2^31-1 (about 59.6 minutes worth)

// TimestampToDIS converts a Unix timestamp to DIS timestamp format
func TimestampToDIS(unixMillis int64) uint32 {
	// Milliseconds since start of hour
	msSinceHour := unixMillis % 3600000
	// Convert to 1/10 millisecond units
	return uint32(msSinceHour * 100)
}

// DISToTimestamp converts DIS timestamp to Unix milliseconds (requires hour reference)
func DISToTimestamp(disTime uint32, hourStartUnix int64) int64 {
	// Convert from 1/10 ms units to milliseconds
	msSinceHour := int64(disTime) / 100
	return hourStartUnix + msSinceHour
}

// DecodeEntityStatePDU deserializes bytes into EntityStatePDU
func DecodeEntityStatePDU(data []byte) (*EntityStatePDU, error) {
	if len(data) < 144 {
		return nil, ErrPDUTooShort
	}
	
	pdu := &EntityStatePDU{}
	
	// Header (12 bytes)
	pdu.ProtocolVersion = data[0]
	pdu.ExerciseID = data[1]
	pdu.PDUType = data[2]
	pdu.ProtocolFamily = data[3]
	pdu.Timestamp = binary.BigEndian.Uint32(data[4:8])
	pdu.Length = binary.BigEndian.Uint16(data[8:10])
	pdu.Padding = binary.BigEndian.Uint16(data[10:12])
	
	// Entity ID (6 bytes)
	pdu.EntityID.SiteID = binary.BigEndian.Uint16(data[12:14])
	pdu.EntityID.ApplicationID = binary.BigEndian.Uint16(data[14:16])
	pdu.EntityID.EntityID = binary.BigEndian.Uint16(data[16:18])
	
	// Force ID (1 byte)
	pdu.ForceID = data[18]
	
	// Number of articulation parameters (1 byte)
	pdu.NumArticulationParams = data[19]
	
	// Entity Type (8 bytes)
	pdu.EntityType = EntityTypeRecord{
		EntityKind: data[20],
		Domain: data[21],
		Country: binary.BigEndian.Uint16(data[22:24]),
		Category: data[24],
		Subcategory: data[25],
		Specific: data[26],
		Extra: data[27],
	}
	
	// Alternative Entity Type (8 bytes)
	pdu.AltEntityType = EntityTypeRecord{
		EntityKind: data[28],
		Domain: data[29],
		Country: binary.BigEndian.Uint16(data[30:32]),
		Category: data[32],
		Subcategory: data[33],
		Specific: data[34],
		Extra: data[35],
	}
	
	// Linear Velocity (12 bytes)
	pdu.LinearVelocity = Vector3Float32{
		X: math.Float32frombits(binary.BigEndian.Uint32(data[36:40])),
		Y: math.Float32frombits(binary.BigEndian.Uint32(data[40:44])),
		Z: math.Float32frombits(binary.BigEndian.Uint32(data[44:48])),
	}
	
	// Location (24 bytes)
	pdu.Location = WorldCoordinate{
		X: math.Float64frombits(binary.BigEndian.Uint64(data[48:56])),
		Y: math.Float64frombits(binary.BigEndian.Uint64(data[56:64])),
		Z: math.Float64frombits(binary.BigEndian.Uint64(data[64:72])),
	}
	
	// Orientation (12 bytes)
	pdu.Orientation = EulerAngles{
		Psi: math.Float32frombits(binary.BigEndian.Uint32(data[72:76])),
		Theta: math.Float32frombits(binary.BigEndian.Uint32(data[76:80])),
		Phi: math.Float32frombits(binary.BigEndian.Uint32(data[80:84])),
	}
	
	// Entity Appearance (4 bytes)
	pdu.Appearance = binary.BigEndian.Uint32(data[84:88])
	
	// Dead Reckoning Algorithm (1 byte)
	pdu.DeadReckoningAlgorithm = data[88]
	
	// Entity Marking (12 bytes)
	pdu.Marking.Characters[0] = data[89]
	copy(pdu.Marking.Characters[:], data[89:100])
	pdu.Marking.CharacterSet = data[100]
	
	// Capabilities (4 bytes)
	pdu.Capabilities = binary.BigEndian.Uint32(data[101:105])
	
	// Variable parameters (variable)
	offset := 112
	numParams := pdu.NumArticulationParams
	for i := uint8(0); i < numParams; i++ {
		if offset+16 > len(data) {
			break
		}
		param := VariableParameter{
			RecordType: data[offset],
		}
		copy(param.Data[:], data[offset+1:offset+16])
		pdu.VariableParams = append(pdu.VariableParams, param)
		offset += 16
	}
	
	return pdu, nil
}

// DecodeFirePDU deserializes bytes into FirePDU
func DecodeFirePDU(data []byte) (*FirePDU, error) {
	if len(data) < 96 {
		return nil, ErrPDUTooShort
	}
	
	pdu := &FirePDU{}
	
	// Header (12 bytes)
	pdu.ProtocolVersion = data[0]
	pdu.ExerciseID = data[1]
	pdu.PDUType = data[2]
	pdu.ProtocolFamily = data[3]
	pdu.Timestamp = binary.BigEndian.Uint32(data[4:8])
	pdu.Length = binary.BigEndian.Uint16(data[8:10])
	
	// Firing Entity ID (6 bytes)
	pdu.FiringEntityID = EntityID{
		SiteID:       binary.BigEndian.Uint16(data[12:14]),
		ApplicationID: binary.BigEndian.Uint16(data[14:16]),
		EntityID:     binary.BigEndian.Uint16(data[16:18]),
	}
	
	// Target Entity ID (6 bytes)
	pdu.TargetEntityID = EntityID{
		SiteID:       binary.BigEndian.Uint16(data[18:20]),
		ApplicationID: binary.BigEndian.Uint16(data[20:22]),
		EntityID:     binary.BigEndian.Uint16(data[22:24]),
	}
	
	// Munition ID (8 bytes)
	pdu.MunitionID = EntityTypeRecord{
		EntityKind: data[24],
		Domain: data[25],
		Country: binary.BigEndian.Uint16(data[26:28]),
		Category: data[28],
		Subcategory: data[29],
		Specific: data[30],
		Extra: data[31],
	}
	
	// Event ID (6 bytes)
	pdu.EventID = EntityID{
		SiteID:       binary.BigEndian.Uint16(data[32:34]),
		ApplicationID: binary.BigEndian.Uint16(data[34:36]),
		EntityID:     binary.BigEndian.Uint16(data[36:38]),
	}
	
	// Fire Mission Index (1 byte)
	pdu.FireMissionIndex = data[38]
	
	// Location (24 bytes)
	pdu.Location = WorldCoordinate{
		X: math.Float64frombits(binary.BigEndian.Uint64(data[40:48])),
		Y: math.Float64frombits(binary.BigEndian.Uint64(data[48:56])),
		Z: math.Float64frombits(binary.BigEndian.Uint64(data[56:64])),
	}
	
	// Velocity (12 bytes)
	pdu.Velocity = Vector3Float32{
		X: math.Float32frombits(binary.BigEndian.Uint32(data[64:68])),
		Y: math.Float32frombits(binary.BigEndian.Uint32(data[68:72])),
		Z: math.Float32frombits(binary.BigEndian.Uint32(data[72:76])),
	}
	
	// Range (4 bytes)
	pdu.Range = math.Float32frombits(binary.BigEndian.Uint32(data[76:80]))
	
	return pdu, nil
}

// DecodeDetonationPDU deserializes bytes into DetonationPDU
func DecodeDetonationPDU(data []byte) (*DetonationPDU, error) {
	if len(data) < 128 {
		return nil, ErrPDUTooShort
	}
	
	pdu := &DetonationPDU{}
	
	// Header (12 bytes)
	pdu.ProtocolVersion = data[0]
	pdu.ExerciseID = data[1]
	pdu.PDUType = data[3]
	pdu.ProtocolFamily = data[4]
	pdu.Timestamp = binary.BigEndian.Uint32(data[4:8])
	pdu.Length = binary.BigEndian.Uint16(data[8:10])
	
	// Firing Entity ID (6 bytes)
	pdu.FiringEntityID = EntityID{
		SiteID:       binary.BigEndian.Uint16(data[12:14]),
		ApplicationID: binary.BigEndian.Uint16(data[14:16]),
		EntityID:     binary.BigEndian.Uint16(data[16:18]),
	}
	
	// Target Entity ID (6 bytes)
	pdu.TargetEntityID = EntityID{
		SiteID:       binary.BigEndian.Uint16(data[18:20]),
		ApplicationID: binary.BigEndian.Uint16(data[20:22]),
		EntityID:     binary.BigEndian.Uint16(data[22:24]),
	}
	
	// Munition ID (8 bytes)
	pdu.MunitionID = EntityTypeRecord{
		EntityKind: data[24],
		Domain: data[25],
		Country: binary.BigEndian.Uint16(data[26:28]),
		Category: data[28],
		Subcategory: data[29],
		Specific: data[30],
		Extra: data[31],
	}
	
	// Event ID (6 bytes)
	pdu.EventID = EntityID{
		SiteID:       binary.BigEndian.Uint16(data[32:34]),
		ApplicationID: binary.BigEndian.Uint16(data[34:36]),
		EntityID:     binary.BigEndian.Uint16(data[36:38]),
	}
	
	// Velocity (12 bytes)
	pdu.Velocity = Vector3Float32{
		X: math.Float32frombits(binary.BigEndian.Uint32(data[48:52])),
		Y: math.Float32frombits(binary.BigEndian.Uint32(data[52:56])),
		Z: math.Float32frombits(binary.BigEndian.Uint32(data[56:60])),
	}
	
	// Location (24 bytes)
	pdu.Location = WorldCoordinate{
		X: math.Float64frombits(binary.BigEndian.Uint64(data[60:68])),
		Y: math.Float64frombits(binary.BigEndian.Uint64(data[68:76])),
		Z: math.Float64frombits(binary.BigEndian.Uint64(data[76:84])),
	}
	
	// Detonation Result (1 byte)
	pdu.DetonationResult = data[84]
	
	return pdu, nil
}
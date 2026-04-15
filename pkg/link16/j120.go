// Package link16 implements Link 16 J-Series message support
// J12.0 is the Mission Assignment message (MIL-STD-6016)
package link16

import (
	"time"
)

// J120Message represents a J12.0 Mission Assignment message
type J120Message struct {
	TrackNumber      uint16    `json:"track_number"`      // Track number
	MissionID        uint16    `json:"mission_id"`        // Mission identifier
	MissionType      uint8     `json:"mission_type"`      // Mission type
	MissionStatus    uint8     `json:"mission_status"`    // Mission status
	Priority         uint8     `json:"priority"`          // Mission priority
	StartTime        time.Time `json:"start_time"`        // Mission start time
	EndTime          time.Time `json:"end_time"`          // Mission end time
	TargetLatitude   float64   `json:"target_latitude"`   // Target latitude
	TargetLongitude  float64   `json:"target_longitude"`  // Target longitude
	TargetAltitude   float64   `json:"target_altitude"`   // Target altitude
	AssignedUnit     uint16    `json:"assigned_unit"`     // Assigned unit ID
	AssignedForce    uint8     `json:"assigned_force"`    // Assigned force
	AssignmentStatus uint8     `json:"assignment_status"` // Assignment status
	Time             time.Time `json:"time"`              // Message time
}

// J120MissionTypes defines mission type codes
const (
	J120MissionUnknown  uint8 = 0  // Unknown mission
	J120MissionCAP      uint8 = 1  // Combat Air Patrol
	J120MissionEscort   uint8 = 2  // Escort
	J120MissionStrike   uint8 = 3  // Strike
	J120MissionSEAD     uint8 = 4  // Suppression of Enemy Air Defense
	J120MissionCAS      uint8 = 5  // Close Air Support
	J120MissionRecon    uint8 = 6  // Reconnaissance
	J120MissionCSAR     uint8 = 7  // Combat Search and Rescue
	J120MissionAWACS    uint8 = 8  // AWACS support
	J120MissionTanker   uint8 = 9  // Tanker support
	J120MissionEW       uint8 = 10 // Electronic Warfare
	J120MissionTraining uint8 = 11 // Training
	J120MissionTest     uint8 = 12 // Test mission
	J120MissionExercise uint8 = 13 // Exercise
	J120MissionOther    uint8 = 14 // Other
)

// J120MissionStatus defines mission status codes
const (
	J120StatusPlanned   uint8 = 0 // Planned
	J120StatusAssigned  uint8 = 1 // Assigned
	J120StatusActive    uint8 = 2 // Active
	J120StatusComplete  uint8 = 3 // Complete
	J120StatusCanceled  uint8 = 4 // Canceled
	J120StatusAborted   uint8 = 5 // Aborted
	J120StatusDelayed   uint8 = 6 // Delayed
	J120StatusSuspended uint8 = 7 // Suspended
)

// J120AssignmentStatus defines assignment status codes
const (
	J120AssignmentPending  uint8 = 0 // Pending
	J120AssignmentAccepted uint8 = 1 // Accepted
	J120AssignmentRejected uint8 = 2 // Rejected
	J120AssignmentComplete uint8 = 3 // Complete
	J120AssignmentFailed   uint8 = 4 // Failed
)

// J120WordCount is the number of words in a J12.0 message
const J120WordCount = 4

// J120Parser handles J12.0 message parsing and generation
type J120Parser struct{}

// NewJ120Parser creates a new J12.0 parser
func NewJ120Parser() *J120Parser {
	return &J120Parser{}
}

// Parse parses a J12.0 message from J-Series words
func (p *J120Parser) Parse(words []uint32) (*J120Message, error) {
	if len(words) < J120WordCount {
		return nil, ErrWordCountMismatch
	}

	msg := &J120Message{}

	// Word 0: Track number (16), Mission ID (16)
	msg.TrackNumber = uint16(words[0] >> 16)
	msg.MissionID = uint16(words[0] & 0xFFFF)

	// Word 1: Mission type (4), status (4), priority (4), force (4), altitude high (8)
	msg.MissionType = uint8(words[1] >> 28)
	msg.MissionStatus = uint8((words[1] >> 24) & 0x0F)
	msg.Priority = uint8((words[1] >> 20) & 0x0F)
	msg.AssignedForce = uint8((words[1] >> 16) & 0x0F)
	altHigh := uint8((words[1] >> 8) & 0xFF)

	// Word 2: Latitude (16), Longitude (16)
	latRaw := int16((words[2] >> 16) & 0xFFFF)
	lonRaw := int16(words[2] & 0xFFFF)
	msg.TargetLatitude = float64(latRaw) * 90.0 / 32767.0
	msg.TargetLongitude = float64(lonRaw) * 180.0 / 32767.0

	// Word 3: Assigned unit (16), Assignment status (4), altitude low (4), reserved (8)
	msg.AssignedUnit = uint16(words[3] >> 16)
	msg.AssignmentStatus = uint8((words[3] >> 12) & 0x0F)
	altLow := uint8((words[3] >> 8) & 0x0F)

	// Combine altitude (high:low bits)
	msg.TargetAltitude = float64(uint16(altHigh)<<4 | uint16(altLow))

	msg.Time = time.Now()

	return msg, nil
}

// Serialize serializes a J12.0 message to J-Series words
func (p *J120Parser) Serialize(msg *J120Message) []uint32 {
	words := make([]uint32, J120WordCount)

	// Word 0: Track number (16), Mission ID (16)
	words[0] = (uint32(msg.TrackNumber) << 16) | uint32(msg.MissionID)

	// Word 1: Mission type (4), status (4), priority (4), force (4), altitude high (8)
	altCombined := uint32(msg.TargetAltitude)
	altHigh := (altCombined >> 4) & 0xFF
	words[1] = (uint32(msg.MissionType&0x0F) << 28) |
		(uint32(msg.MissionStatus&0x0F) << 24) |
		(uint32(msg.Priority&0x0F) << 20) |
		(uint32(msg.AssignedForce&0x0F) << 16) |
		uint32(altHigh)<<8

	// Word 2: Latitude (16), Longitude (16)
	latRaw := int16(msg.TargetLatitude * 32767.0 / 90.0)
	lonRaw := int16(msg.TargetLongitude * 32767.0 / 180.0)
	words[2] = (uint32(uint16(latRaw)) << 16) |
		uint32(uint16(lonRaw))

	// Word 3: Assigned unit (16), Assignment status (4), altitude low (4), reserved (8)
	altLow := altCombined & 0x0F
	words[3] = (uint32(msg.AssignedUnit) << 16) |
		(uint32(msg.AssignmentStatus&0x0F) << 12) |
		uint32(altLow)<<8

	return words
}

// GetMissionTypeString returns string representation of mission type
func GetMissionTypeString(missionType uint8) string {
	switch missionType {
	case J120MissionUnknown:
		return "UNKNOWN"
	case J120MissionCAP:
		return "CAP"
	case J120MissionEscort:
		return "ESCORT"
	case J120MissionStrike:
		return "STRIKE"
	case J120MissionSEAD:
		return "SEAD"
	case J120MissionCAS:
		return "CAS"
	case J120MissionRecon:
		return "RECON"
	case J120MissionCSAR:
		return "CSAR"
	case J120MissionAWACS:
		return "AWACS"
	case J120MissionTanker:
		return "TANKER"
	case J120MissionEW:
		return "EW"
	case J120MissionTraining:
		return "TRAINING"
	case J120MissionTest:
		return "TEST"
	case J120MissionExercise:
		return "EXERCISE"
	case J120MissionOther:
		return "OTHER"
	default:
		return "UNKNOWN"
	}
}

// GetMissionStatusString returns string representation of mission status
func GetMissionStatusString(status uint8) string {
	switch status {
	case J120StatusPlanned:
		return "PLANNED"
	case J120StatusAssigned:
		return "ASSIGNED"
	case J120StatusActive:
		return "ACTIVE"
	case J120StatusComplete:
		return "COMPLETE"
	case J120StatusCanceled:
		return "CANCELED"
	case J120StatusAborted:
		return "ABORTED"
	case J120StatusDelayed:
		return "DELAYED"
	case J120StatusSuspended:
		return "SUSPENDED"
	default:
		return "UNKNOWN"
	}
}

// GetAssignmentStatusString returns string representation of assignment status
func GetAssignmentStatusString(status uint8) string {
	switch status {
	case J120AssignmentPending:
		return "PENDING"
	case J120AssignmentAccepted:
		return "ACCEPTED"
	case J120AssignmentRejected:
		return "REJECTED"
	case J120AssignmentComplete:
		return "COMPLETE"
	case J120AssignmentFailed:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// J120Builder helps build J12.0 messages
type J120Builder struct {
	msg *J120Message
}

// NewJ120Builder creates a new J12.0 builder
func NewJ120Builder() *J120Builder {
	return &J120Builder{
		msg: &J120Message{
			Time: time.Now(),
		},
	}
}

// SetTrackNumber sets track number
func (b *J120Builder) SetTrackNumber(tn uint16) *J120Builder {
	b.msg.TrackNumber = tn
	return b
}

// SetMissionID sets mission ID
func (b *J120Builder) SetMissionID(id uint16) *J120Builder {
	b.msg.MissionID = id
	return b
}

// SetMissionType sets mission type
func (b *J120Builder) SetMissionType(missionType uint8) *J120Builder {
	b.msg.MissionType = missionType
	return b
}

// SetStatus sets mission status
func (b *J120Builder) SetStatus(status uint8) *J120Builder {
	b.msg.MissionStatus = status
	return b
}

// SetPriority sets priority
func (b *J120Builder) SetPriority(priority uint8) *J120Builder {
	b.msg.Priority = priority
	return b
}

// SetTarget sets target position
func (b *J120Builder) SetTarget(lat, lon, alt float64) *J120Builder {
	b.msg.TargetLatitude = lat
	b.msg.TargetLongitude = lon
	b.msg.TargetAltitude = alt
	return b
}

// SetAssignedUnit sets assigned unit
func (b *J120Builder) SetAssignedUnit(unit uint16, force uint8) *J120Builder {
	b.msg.AssignedUnit = unit
	b.msg.AssignedForce = force
	return b
}

// SetAssignmentStatus sets assignment status
func (b *J120Builder) SetAssignmentStatus(status uint8) *J120Builder {
	b.msg.AssignmentStatus = status
	return b
}

// SetTimes sets start and end times
func (b *J120Builder) SetTimes(start, end time.Time) *J120Builder {
	b.msg.StartTime = start
	b.msg.EndTime = end
	return b
}

// Build builds the J12.0 message
func (b *J120Builder) Build() *J120Message {
	return b.msg
}

// J120Mission represents a mission from J12.0 data
type J120Mission struct {
	MissionID      uint32     `json:"mission_id"`
	MissionType    string     `json:"mission_type"`
	MissionStatus  string     `json:"mission_status"`
	TargetPosition [3]float64 `json:"target_position"` // lat, lon, alt
	AssignedUnit   uint32     `json:"assigned_unit"`
	AssignedForce  string     `json:"assigned_force"`
	Priority       uint8      `json:"priority"`
	StartTime      time.Time  `json:"start_time"`
	EndTime        time.Time  `json:"end_time"`
	LastUpdate     time.Time  `json:"last_update"`
}

// ToMission converts J120Message to J120Mission
func (msg *J120Message) ToMission() *J120Mission {
	return &J120Mission{
		MissionID:      uint32(msg.MissionID),
		MissionType:    GetMissionTypeString(msg.MissionType),
		MissionStatus:  GetMissionStatusString(msg.MissionStatus),
		TargetPosition: [3]float64{msg.TargetLatitude, msg.TargetLongitude, msg.TargetAltitude},
		AssignedUnit:   uint32(msg.AssignedUnit),
		AssignedForce:  GetIdentityString(msg.AssignedForce),
		Priority:       msg.Priority,
		StartTime:      msg.StartTime,
		EndTime:        msg.EndTime,
		LastUpdate:     msg.Time,
	}
}

// J120Stats holds J12.0 message statistics
type J120Stats struct {
	TotalMissions     uint64    `json:"total_missions"`
	ActiveMissions    uint64    `json:"active_missions"`
	CompletedMissions uint64    `json:"completed_missions"`
	AssignedMissions  uint64    `json:"assigned_missions"`
	LastUpdateTime    time.Time `json:"last_update_time"`
}

// Package link16 implements Link 16 J-Series message support
// J7.0 is the Track Management message (MIL-STD-6016)
package link16

import (
	"time"
)

// J70Message represents a J7.0 Track Management message
type J70Message struct {
	TrackNumber      uint16    `json:"track_number"`      // Track number
	TrackStatus      uint8     `json:"track_status"`       // Track status (0-15)
	TrackQuality     uint8     `json:"track_quality"`      // Track quality (0-15)
	TrackIdentity    uint8     `json:"track_identity"`     // Identity code
	TrackAugment     uint8     `json:"track_augment"`      // Augmentation
	ForceID          uint8     `json:"force_id"`           // Force ID
	Environment      uint8     `json:"environment"`        // Environment
	ActionCode       uint8     `json:"action_code"`        // Track action code
	SourceTrack      uint16    `json:"source_track"`       // Source track number
	TargetTrack      uint16    `json:"target_track"`      // Target track number
	CorrelationCode  uint8     `json:"correlation_code"`   // Correlation code
	Time             time.Time `json:"time"`              // Time of message
}

// J70ActionCodes defines track action codes
const (
	J70ActionNewTrack       uint8 = 0  // New track
	J70ActionUpdate         uint8 = 1  // Update track
	J70ActionDelete         uint8 = 2  // Delete track
	J70ActionCorrelate      uint8 = 3  // Correlate tracks
	J70ActionDecorrelate    uint8 = 4  // Decorrelate tracks
	J70ActionMerge          uint8 = 5  // Merge tracks
	J70ActionSplit          uint8 = 6  // Split track
	J70ActionChangeID       uint8 = 7  // Change track ID
	J70ActionDropTrack      uint8 = 8  // Drop track
	J70ActionPromote        uint8 = 9  // Promote track
	J70ActionDegrade        uint8 = 10 // Degrade track
	J70ActionRequestInfo    uint8 = 11 // Request information
	J70ActionProvideInfo    uint8 = 12 // Provide information
	J70ActionConfirmID      uint8 = 13 // Confirm identity
	J70ActionChangeType     uint8 = 14 // Change track type
	J70ActionEmergency      uint8 = 15 // Emergency action
)

// J70TrackStatus defines track status codes
const (
	J70StatusDropped      uint8 = 0  // Track dropped
	J70StatusInitiating   uint8 = 1  // Track initiating
	J70StatusTentative    uint8 = 2  // Tentative track
	J70StatusConfirmed    uint8 = 3  // Confirmed track
	J70StatusCoasting     uint8 = 4  // Track coasting
	J70StatusPredicted    uint8 = 5  // Predicted track
	J70StatusLost         uint8 = 6  // Track lost
	J70StatusUnknown      uint8 = 7  // Unknown status
)

// J70Parser handles J7.0 message parsing and generation
type J70Parser struct{}

// NewJ70Parser creates a new J7.0 parser
func NewJ70Parser() *J70Parser {
	return &J70Parser{}
}

// Parse parses a J7.0 message from J-Series words
func (p *J70Parser) Parse(words []uint32) (*J70Message, error) {
	if len(words) < 2 {
		return nil, ErrWordCountMismatch
	}
	
	msg := &J70Message{}
	
	// Word 0: Track number, status, quality, identity
	msg.TrackNumber = uint16(words[0] >> 16)
	msg.TrackStatus = uint8((words[0] >> 12) & 0x0F)
	msg.TrackQuality = uint8((words[0] >> 8) & 0x0F)
	msg.TrackIdentity = uint8((words[0] >> 4) & 0x0F)
	msg.ForceID = uint8(words[0] & 0x0F)
	
	// Word 1: Action code, source, target
	msg.ActionCode = uint8(words[1] >> 28)
	msg.SourceTrack = uint16((words[1] >> 12) & 0xFFFF)
	msg.TargetTrack = uint16(words[1] & 0x0FFF)
	msg.Environment = uint8((words[1] >> 24) & 0x0F)
	
	msg.Time = time.Now()
	
	return msg, nil
}

// Serialize serializes a J7.0 message to J-Series words
func (p *J70Parser) Serialize(msg *J70Message) []uint32 {
	words := make([]uint32, 2)
	
	// Word 0: Track number, status, quality, identity
	words[0] = (uint32(msg.TrackNumber) << 16) |
		(uint32(msg.TrackStatus&0x0F) << 12) |
		(uint32(msg.TrackQuality&0x0F) << 8) |
		(uint32(msg.TrackIdentity&0x0F) << 4) |
		(uint32(msg.ForceID) & 0x0F)
	
	// Word 1: Action code, source, target
	words[1] = (uint32(msg.ActionCode&0x0F) << 28) |
		(uint32(msg.Environment&0x0F) << 24) |
		(uint32(msg.SourceTrack&0xFFFF) << 12) |
		(uint32(msg.TargetTrack) & 0x0FFF)
	
	return words
}

// GetActionString returns string representation of action code
func GetActionString(action uint8) string {
	switch action {
	case J70ActionNewTrack:
		return "NEW_TRACK"
	case J70ActionUpdate:
		return "UPDATE"
	case J70ActionDelete:
		return "DELETE"
	case J70ActionCorrelate:
		return "CORRELATE"
	case J70ActionDecorrelate:
		return "DECORRELATE"
	case J70ActionMerge:
		return "MERGE"
	case J70ActionSplit:
		return "SPLIT"
	case J70ActionChangeID:
		return "CHANGE_ID"
	case J70ActionDropTrack:
		return "DROP_TRACK"
	case J70ActionPromote:
		return "PROMOTE"
	case J70ActionDegrade:
		return "DEGRADE"
	case J70ActionRequestInfo:
		return "REQUEST_INFO"
	case J70ActionProvideInfo:
		return "PROVIDE_INFO"
	case J70ActionConfirmID:
		return "CONFIRM_ID"
	case J70ActionChangeType:
		return "CHANGE_TYPE"
	case J70ActionEmergency:
		return "EMERGENCY"
	default:
		return "UNKNOWN"
	}
}

// GetStatusString returns string representation of status
func GetStatusString(status uint8) string {
	switch status {
	case J70StatusDropped:
		return "DROPPED"
	case J70StatusInitiating:
		return "INITIATING"
	case J70StatusTentative:
		return "TENTATIVE"
	case J70StatusConfirmed:
		return "CONFIRMED"
	case J70StatusCoasting:
		return "COASTING"
	case J70StatusPredicted:
		return "PREDICTED"
	case J70StatusLost:
		return "LOST"
	case J70StatusUnknown:
		return "UNKNOWN"
	default:
		return "UNKNOWN"
	}
}

// J70Builder helps build J7.0 messages
type J70Builder struct {
	msg *J70Message
}

// NewJ70Builder creates a new J7.0 builder
func NewJ70Builder() *J70Builder {
	return &J70Builder{
		msg: &J70Message{
			Time: time.Now(),
		},
	}
}

// SetTrackNumber sets track number
func (b *J70Builder) SetTrackNumber(tn uint16) *J70Builder {
	b.msg.TrackNumber = tn
	return b
}

// SetStatus sets track status
func (b *J70Builder) SetStatus(status uint8) *J70Builder {
	b.msg.TrackStatus = status
	return b
}

// SetQuality sets track quality
func (b *J70Builder) SetQuality(quality uint8) *J70Builder {
	b.msg.TrackQuality = quality
	return b
}

// SetIdentity sets track identity
func (b *J70Builder) SetIdentity(identity uint8) *J70Builder {
	b.msg.TrackIdentity = identity
	return b
}

// SetForce sets force ID
func (b *J70Builder) SetForce(force uint8) *J70Builder {
	b.msg.ForceID = force
	return b
}

// SetEnvironment sets environment
func (b *J70Builder) SetEnvironment(env uint8) *J70Builder {
	b.msg.Environment = env
	return b
}

// SetAction sets action code
func (b *J70Builder) SetAction(action uint8) *J70Builder {
	b.msg.ActionCode = action
	return b
}

// SetSource sets source track
func (b *J70Builder) SetSource(source uint16) *J70Builder {
	b.msg.SourceTrack = source
	return b
}

// SetTarget sets target track
func (b *J70Builder) SetTarget(target uint16) *J70Builder {
	b.msg.TargetTrack = target
	return b
}

// Build builds the J7.0 message
func (b *J70Builder) Build() *J70Message {
	return b.msg
}

// J70Action represents a track management action
type J70Action struct {
	Action    uint8     `json:"action"`
	Source    uint16    `json:"source"`
	Target    uint16    `json:"target"`
	Timestamp time.Time `json:"timestamp"`
}

// NewTrack creates a new track action
func NewTrackAction(trackNum uint16) *J70Action {
	return &J70Action{
		Action:    J70ActionNewTrack,
		Source:    0,
		Target:    trackNum,
		Timestamp: time.Now(),
	}
}

// UpdateTrack creates an update track action
func UpdateTrackAction(trackNum uint16) *J70Action {
	return &J70Action{
		Action:    J70ActionUpdate,
		Source:    0,
		Target:    trackNum,
		Timestamp: time.Now(),
	}
}

// DeleteTrack creates a delete track action
func DeleteTrackAction(trackNum uint16) *J70Action {
	return &J70Action{
		Action:    J70ActionDelete,
		Source:    0,
		Target:    trackNum,
		Timestamp: time.Now(),
	}
}

// CorrelateTracks creates a correlate action
func CorrelateTracksAction(source, target uint16) *J70Action {
	return &J70Action{
		Action:    J70ActionCorrelate,
		Source:    source,
		Target:    target,
		Timestamp: time.Now(),
	}
}

// MergeTracks creates a merge action
func MergeTracksAction(source, target uint16) *J70Action {
	return &J70Action{
		Action:    J70ActionMerge,
		Source:    source,
		Target:    target,
		Timestamp: time.Now(),
	}
}

// J70Stats holds J7.0 message statistics
type J70Stats struct {
	TotalMessages    uint64 `json:"total_messages"`
	NewTracks       uint64 `json:"new_tracks"`
	Updates         uint64 `json:"updates"`
	Deletes         uint64 `json:"deletes"`
	Correlations    uint64 `json:"correlations"`
	Merges          uint64 `json:"merges"`
	LastUpdateTime  time.Time `json:"last_update_time"`
}

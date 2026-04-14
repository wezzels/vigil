// Package link16 implements Link 16 J-Series message support
// J3.2 is the Air Track message (MIL-STD-6016)
package link16

import (
	"time"
)

// Errors
var (
	ErrWordCountMismatch = &J32Error{Code: "WORD_COUNT_MISMATCH", Message: "word count mismatch"}
)

// J32Error represents a J3.2 error
type J32Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *J32Error) Error() string {
	return e.Message
}

// J32Message represents a J3.2 Air Track message
type J32Message struct {
	TrackNumber     uint16    `json:"track_number"`     // Track number
	TrackQuality    uint8     `json:"track_quality"`    // Track quality (0-15)
	TrackIdentity   uint8     `json:"track_identity"`   // Identity (0-15)
	TrackAugment    uint8     `json:"track_augment"`    // Augmentation
	Latitude        float64   `json:"latitude"`         // Degrees (-90 to +90)
	Longitude       float64   `json:"longitude"`        // Degrees (-180 to +180)
	Altitude        float64   `json:"altitude"`         // Meters
	Speed           float64   `json:"speed"`            // Meters per second
	Heading         float64   `json:"heading"`          // Degrees (0-360)
	Time            time.Time `json:"time"`            // Time of track
	Force           uint8     `json:"force"`           // Force ID
	Environment     uint8     `json:"environment"`     // Environment (Air, Surface, etc.)
	TrackType       uint8     `json:"track_type"`       // Track type
}

// J32Constants defines J3.2 field constants
const (
	J32WordCount       = 3
	J32LatitudeScale   = 90.0 / 32767.0   // Scale for 15-bit latitude (-90 to +90)
	J32LongitudeScale  = 180.0 / 32767.0   // Scale for 15-bit longitude (-180 to +180)
	J32AltitudeScale    = 1.0              // Scale for 16-bit altitude (meters)
	J32SpeedScale       = 10.0             // Scale for 11-bit speed (dm/s)
	J32HeadingScale     = 360.0 / 512.0    // Scale for 9-bit heading (degrees)
)

// Identity codes (MIL-STD-6016)
const (
	IdentityPending     uint8 = 0
	IdentityUnknown      uint8 = 1
	IdentityAssumedFriend uint8 = 2
	IdentityFriend       uint8 = 3
	IdentityNeutral      uint8 = 4
	IdentitySuspect      uint8 = 5
	IdentityHostile      uint8 = 6
)

// Environment codes
const (
	EnvAir     uint8 = 0
	EnvSurface uint8 = 1
	EnvSubsurface uint8 = 2
	EnvLand    uint8 = 3
)

// J32Parser handles J3.2 message parsing and generation
type J32Parser struct{}

// NewJ32Parser creates a new J3.2 parser
func NewJ32Parser() *J32Parser {
	return &J32Parser{}
}

// Parse parses a J3.2 message from J-Series words
func (p *J32Parser) Parse(words []uint32) (*J32Message, error) {
	if len(words) < J32WordCount {
		return nil, ErrWordCountMismatch
	}
	
	msg := &J32Message{}
	
	// Word 0: Track number and quality
	msg.TrackNumber = uint16(words[0] >> 16)
	msg.TrackQuality = uint8((words[0] >> 12) & 0x0F)
	msg.TrackIdentity = uint8((words[0] >> 8) & 0x0F)
	msg.Force = uint8((words[0] >> 4) & 0x0F)
	msg.Environment = uint8(words[0] & 0x0F)
	
	// Word 1: Position (signed 16-bit values)
	// Latitude: -90 to +90 degrees, scaled to 16-bit
	// Longitude: -180 to +180 degrees, scaled to 16-bit
	latRaw := int16(words[1] >> 16)
	lonRaw := int16(words[1] & 0xFFFF)
	msg.Latitude = float64(latRaw) * 90.0 / 32767.0
	msg.Longitude = float64(lonRaw) * 180.0 / 32767.0
	
	// Word 2: Altitude, speed, heading
	alt := uint16((words[2] >> 16) & 0xFFFF)
	speed := uint16((words[2] >> 5) & 0x07FF)
	heading := uint16(words[2] & 0x1FF)
	
	msg.Altitude = float64(alt)
	msg.Speed = float64(speed) * J32SpeedScale
	msg.Heading = float64(heading) * J32HeadingScale
	
	msg.Time = time.Now()
	
	return msg, nil
}

// Serialize serializes a J3.2 message to J-Series words
func (p *J32Parser) Serialize(msg *J32Message) []uint32 {
	words := make([]uint32, J32WordCount)
	
	// Word 0: Track number and quality
	words[0] = (uint32(msg.TrackNumber) << 16) |
		(uint32(msg.TrackQuality&0x0F) << 12) |
		(uint32(msg.TrackIdentity&0x0F) << 8) |
		(uint32(msg.Force&0x0F) << 4) |
		(uint32(msg.Environment) & 0x0F)
	
	// Word 1: Position
	// Latitude: -90 to +90, scale to signed 16-bit
	// Longitude: -180 to +180, scale to signed 16-bit
	latRaw := int16(msg.Latitude * 32767.0 / 90.0)
	lonRaw := int16(msg.Longitude * 32767.0 / 180.0)
	words[1] = (uint32(uint16(latRaw)) << 16) | uint32(uint16(lonRaw))
	
	// Word 2: Altitude, speed, heading
	alt := uint16(msg.Altitude)
	speed := uint16(msg.Speed / J32SpeedScale)
	heading := uint16(msg.Heading / J32HeadingScale)
	words[2] = (uint32(alt) << 16) |
		(uint32(speed&0x07FF) << 5) |
		(uint32(heading) & 0x1FF)
	
	return words
}

// GetIdentityString returns string representation of identity
func GetIdentityString(identity uint8) string {
	switch identity {
	case IdentityPending:
		return "PENDING"
	case IdentityUnknown:
		return "UNKNOWN"
	case IdentityAssumedFriend:
		return "ASSUMED_FRIEND"
	case IdentityFriend:
		return "FRIEND"
	case IdentityNeutral:
		return "NEUTRAL"
	case IdentitySuspect:
		return "SUSPECT"
	case IdentityHostile:
		return "HOSTILE"
	default:
		return "UNKNOWN"
	}
}

// GetEnvironmentString returns string representation of environment
func GetEnvironmentString(env uint8) string {
	switch env {
	case EnvAir:
		return "AIR"
	case EnvSurface:
		return "SURFACE"
	case EnvSubsurface:
		return "SUBSURFACE"
	case EnvLand:
		return "LAND"
	default:
		return "UNKNOWN"
	}
}

// J32Builder helps build J3.2 messages
type J32Builder struct {
	msg *J32Message
}

// NewJ32Builder creates a new J3.2 builder
func NewJ32Builder() *J32Builder {
	return &J32Builder{
		msg: &J32Message{
			Time: time.Now(),
		},
	}
}

// SetTrackNumber sets track number
func (b *J32Builder) SetTrackNumber(tn uint16) *J32Builder {
	b.msg.TrackNumber = tn
	return b
}

// SetPosition sets position
func (b *J32Builder) SetPosition(lat, lon, alt float64) *J32Builder {
	b.msg.Latitude = lat
	b.msg.Longitude = lon
	b.msg.Altitude = alt
	return b
}

// SetVelocity sets velocity
func (b *J32Builder) SetVelocity(speed, heading float64) *J32Builder {
	b.msg.Speed = speed
	b.msg.Heading = heading
	return b
}

// SetIdentity sets identity
func (b *J32Builder) SetIdentity(identity uint8) *J32Builder {
	b.msg.TrackIdentity = identity
	return b
}

// SetForce sets force
func (b *J32Builder) SetForce(force uint8) *J32Builder {
	b.msg.Force = force
	return b
}

// SetEnvironment sets environment
func (b *J32Builder) SetEnvironment(env uint8) *J32Builder {
	b.msg.Environment = env
	return b
}

// SetQuality sets track quality
func (b *J32Builder) SetQuality(quality uint8) *J32Builder {
	b.msg.TrackQuality = quality
	return b
}

// Build builds the J3.2 message
func (b *J32Builder) Build() *J32Message {
	return b.msg
}

// J32Track represents a track from J3.2 data
type J32Track struct {
	TrackNumber  uint32    `json:"track_number"`
	Position     [3]float64 `json:"position"` // lat, lon, alt (deg, deg, m)
	Velocity     [2]float64 `json:"velocity"` // speed (m/s), heading (deg)
	Identity     string    `json:"identity"`
	Force        string    `json:"force"`
	Environment  string    `json:"environment"`
	Quality      uint8     `json:"quality"`
	LastUpdate   time.Time `json:"last_update"`
}

// ToTrack converts J32Message to J32Track
func (msg *J32Message) ToTrack() *J32Track {
	return &J32Track{
		TrackNumber: uint32(msg.TrackNumber),
		Position:     [3]float64{msg.Latitude, msg.Longitude, msg.Altitude},
		Velocity:     [2]float64{msg.Speed, msg.Heading},
		Identity:     GetIdentityString(msg.TrackIdentity),
		Force:        GetIdentityString(msg.Force),
		Environment:  GetEnvironmentString(msg.Environment),
		Quality:      msg.TrackQuality,
		LastUpdate:   msg.Time,
	}
}

// FromTrack converts J32Track to J32Message
func FromTrack(track *J32Track) *J32Message {
	identity := IdentityUnknown
	for i, id := range []uint8{IdentityPending, IdentityUnknown, IdentityAssumedFriend,
		IdentityFriend, IdentityNeutral, IdentitySuspect, IdentityHostile} {
		if GetIdentityString(id) == track.Identity {
			identity = uint8(i)
			break
		}
	}
	
	env := EnvAir
	for i, e := range []uint8{EnvAir, EnvSurface, EnvSubsurface, EnvLand} {
		if GetEnvironmentString(e) == track.Environment {
			env = uint8(i)
			break
		}
	}
	
	return &J32Message{
		TrackNumber:   uint16(track.TrackNumber),
		TrackQuality:  track.Quality,
		TrackIdentity: identity,
		Latitude:      track.Position[0],
		Longitude:     track.Position[1],
		Altitude:      track.Position[2],
		Speed:         track.Velocity[0],
		Heading:       track.Velocity[1],
		Time:          track.LastUpdate,
		Force:         identity, // Use same as identity
		Environment:   env,
	}
}

// J32Stats holds J3.2 message statistics
type J32Stats struct {
	TotalMessages   uint64 `json:"total_messages"`
	ValidMessages   uint64 `json:"valid_messages"`
	InvalidMessages uint64 `json:"invalid_messages"`
	AirTracks      uint64 `json:"air_tracks"`
	SurfaceTracks  uint64 `json:"surface_tracks"`
	LastUpdateTime time.Time `json:"last_update_time"`
}
// Package jseries implements Link 16 J-Series message parsing and generation
// Link 16 is defined in MIL-STD-6016
package jseries

import (
	"encoding/binary"
)

// JSeriesHeader represents a J-Series message header
type JSeriesHeader struct {
	MessageNumber uint16 `json:"message_number"` // Jn.m number
	Submessage    uint8  `json:"submessage"`     // Submessage identifier
	WordCount     uint8  `json:"word_count"`     // Number of data words
	Priority      uint8  `json:"priority"`       // Message priority
	TimeSlot      uint8  `json:"time_slot"`      // Time slot assignment
	Security      uint8  `json:"security"`       // Security classification
	Compression   uint8  `json:"compression"`    // Compression indicator
}

// JSeriesMessage represents a complete J-Series message
type JSeriesMessage struct {
	Header JSeriesHeader `json:"header"`
	Words  []uint32      `json:"words"` // 70-bit words
	Data   []byte        `json:"data"`  // Raw data
	Valid  bool          `json:"valid"`
}

// JMessageNumbers defines standard J-Series message numbers
const (
	J0_0  = 0*16 + 0  // J0.0 - Initial Entry
	J0_1  = 0*16 + 1  // J0.1 - Network Management
	J0_2  = 0*16 + 2  // J0.2 - Time Quality
	J2_0  = 2*16 + 0  // J2.0 - Air Track
	J2_2  = 2*16 + 2  // J2.2 - Surface Track
	J2_3  = 2*16 + 3  // J2.3 - Subsurface Track
	J2_4  = 2*16 + 4  // J2.4 - Land Track
	J2_5  = 2*16 + 5  // J2.5 - Reference Point
	J3_0  = 3*16 + 0  // J3.0 - Air Track (continuation)
	J3_2  = 3*16 + 2  // J3.2 - Air Track
	J3_3  = 3*16 + 3  // J3.3 - Surface Track
	J3_4  = 3*16 + 4  // J3.4 - Subsurface Track
	J3_5  = 3*16 + 5  // J3.5 - Land Track
	J7_0  = 7*16 + 0  // J7.0 - Track Management
	J7_1  = 7*16 + 1  // J7.1 - Track Correlation
	J7_2  = 7*16 + 2  // J7.2 - Identification
	J12_0 = 12*16 + 0 // J12.0 - Mission Assignment
	J12_1 = 12*16 + 1 // J12.1 - Engagement Coordination
	J13_0 = 13*16 + 0 // J13.0 - Weapon Engagement
	J13_2 = 13*16 + 2 // J13.2 - Weapon Engagement (continuation)
)

// WordBitLength is the number of bits in a J-Series word
const WordBitLength = 70

// WordByteLength is the number of bytes in a J-Series word
const WordByteLength = 9 // 70 bits = 9 bytes (with padding)

// Parser handles J-Series message parsing
type Parser struct {
	wordBuffer []byte
}

// NewParser creates a new J-Series parser
func NewParser() *Parser {
	return &Parser{
		wordBuffer: make([]byte, WordByteLength),
	}
}

// ParseHeader parses a J-Series message header
func (p *Parser) ParseHeader(data []byte) (JSeriesHeader, error) {
	if len(data) < 4 {
		return JSeriesHeader{}, ErrDataTooShort
	}

	var header JSeriesHeader

	// First 2 bytes: Message number and submessage
	msgWord := binary.BigEndian.Uint16(data[0:2])
	header.MessageNumber = (msgWord >> 4) & 0xFFF
	header.Submessage = uint8(msgWord & 0x0F)

	// Third byte: Word count and priority
	header.WordCount = (data[2] >> 4) & 0x0F
	header.Priority = data[2] & 0x0F

	// Fourth byte: Time slot, security, compression
	header.TimeSlot = (data[3] >> 5) & 0x07
	header.Security = (data[3] >> 2) & 0x07
	header.Compression = data[3] & 0x03

	return header, nil
}

// ParseMessage parses a complete J-Series message
func (p *Parser) ParseMessage(data []byte) (JSeriesMessage, error) {
	if len(data) < 4 {
		return JSeriesMessage{}, ErrDataTooShort
	}

	header, err := p.ParseHeader(data)
	if err != nil {
		return JSeriesMessage{}, err
	}

	// Calculate expected data length
	dataOffset := 4
	dataLength := int(header.WordCount) * WordByteLength

	if len(data) < dataOffset+dataLength {
		return JSeriesMessage{}, ErrDataTooShort
	}

	msg := JSeriesMessage{
		Header: header,
		Valid:  true,
	}

	// Parse 70-bit words
	for i := 0; i < int(header.WordCount); i++ {
		wordStart := dataOffset + i*WordByteLength
		word := p.parseWord(data[wordStart : wordStart+WordByteLength])
		msg.Words = append(msg.Words, word)
	}

	// Extract raw data
	msg.Data = make([]byte, dataLength)
	copy(msg.Data, data[dataOffset:dataOffset+dataLength])

	return msg, nil
}

// parseWord parses a 70-bit word from 9 bytes
func (p *Parser) parseWord(data []byte) uint32 {
	// 70-bit word packed in 9 bytes
	// Extract 32-bit value from the most significant bits
	var word uint32
	for i := 0; i < 4; i++ {
		word = (word << 8) | uint32(data[i])
	}
	return word >> 2 // Adjust for 70-bit alignment
}

// SerializeHeader serializes a J-Series header to bytes
func (p *Parser) SerializeHeader(header JSeriesHeader) []byte {
	data := make([]byte, 4)

	// First 2 bytes: Message number and submessage
	msgWord := (header.MessageNumber << 4) | uint16(header.Submessage)
	binary.BigEndian.PutUint16(data[0:2], msgWord)

	// Third byte: Word count and priority
	data[2] = ((header.WordCount & 0x0F) << 4) | (header.Priority & 0x0F)

	// Fourth byte: Time slot, security, compression
	data[3] = ((header.TimeSlot & 0x07) << 5) | ((header.Security & 0x07) << 2) | (header.Compression & 0x03)

	return data
}

// SerializeMessage serializes a J-Series message to bytes
func (p *Parser) SerializeMessage(msg JSeriesMessage) []byte {
	headerBytes := p.SerializeHeader(msg.Header)

	totalLength := len(headerBytes) + len(msg.Words)*WordByteLength
	data := make([]byte, totalLength)

	copy(data[0:len(headerBytes)], headerBytes)

	// Serialize words
	for i, word := range msg.Words {
		p.serializeWord(word, data[len(headerBytes)+i*WordByteLength:])
	}

	return data
}

// serializeWord serializes a 32-bit value to a 70-bit word
func (p *Parser) serializeWord(word uint32, data []byte) {
	// Pack 32-bit value into 70-bit word
	word <<= 2 // Adjust for 70-bit alignment
	for i := 0; i < 4; i++ {
		data[i] = byte((word >> (24 - i*8)) & 0xFF)
	}
	// Fill remaining bytes with zeros
	for i := 4; i < WordByteLength; i++ {
		data[i] = 0
	}
}

// GetJMessageType returns the J message type string
func GetJMessageType(msgNum uint16) string {
	switch msgNum {
	case J0_0:
		return "J0.0 (Initial Entry)"
	case J0_1:
		return "J0.1 (Network Management)"
	case J0_2:
		return "J0.2 (Time Quality)"
	case J2_0:
		return "J2.0 (Air Track)"
	case J2_2:
		return "J2.2 (Surface Track)"
	case J2_3:
		return "J2.3 (Subsurface Track)"
	case J2_4:
		return "J2.4 (Land Track)"
	case J2_5:
		return "J2.5 (Reference Point)"
	case J3_0:
		return "J3.0 (Air Track Continuation)"
	case J3_2:
		return "J3.2 (Air Track)"
	case J3_3:
		return "J3.3 (Surface Track)"
	case J3_4:
		return "J3.4 (Subsurface Track)"
	case J3_5:
		return "J3.5 (Land Track)"
	case J7_0:
		return "J7.0 (Track Management)"
	case J7_1:
		return "J7.1 (Track Correlation)"
	case J7_2:
		return "J7.2 (Identification)"
	case J12_0:
		return "J12.0 (Mission Assignment)"
	case J12_1:
		return "J12.1 (Engagement Coordination)"
	case J13_0:
		return "J13.0 (Weapon Engagement)"
	case J13_2:
		return "J13.2 (Weapon Engagement Cont.)"
	default:
		return "Unknown"
	}
}

// ValidateMessage validates a J-Series message
func (p *Parser) ValidateMessage(msg JSeriesMessage) error {
	// Check message number
	if msg.Header.MessageNumber > 15*16+15 {
		return ErrInvalidMessageNumber
	}

	// Check word count
	if msg.Header.WordCount > 8 {
		return ErrInvalidWordCount
	}

	// Check priority
	if msg.Header.Priority > 15 {
		return ErrInvalidPriority
	}

	// Check word count matches
	if len(msg.Words) != int(msg.Header.WordCount) {
		return ErrWordCountMismatch
	}

	return nil
}

// Errors
var (
	ErrDataTooShort         = &JSeriesError{Code: "DATA_TOO_SHORT", Message: "data too short"}
	ErrInvalidMessageNumber = &JSeriesError{Code: "INVALID_MESSAGE_NUMBER", Message: "invalid message number"}
	ErrInvalidWordCount     = &JSeriesError{Code: "INVALID_WORD_COUNT", Message: "invalid word count"}
	ErrInvalidPriority      = &JSeriesError{Code: "INVALID_PRIORITY", Message: "invalid priority"}
	ErrWordCountMismatch    = &JSeriesError{Code: "WORD_COUNT_MISMATCH", Message: "word count mismatch"}
)

// JSeriesError represents a J-Series error
type JSeriesError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *JSeriesError) Error() string {
	return e.Message
}

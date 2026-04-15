package mode

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// ReplayManager handles data replay from recorded files
type ReplayManager struct {
	config        *ReplayConfig
	file          *os.File
	reader        *bufio.Reader
	gzipReader    *gzip.Reader
	records       []ReplayRecord
	currentIdx    int
	playbackMu    sync.Mutex
	startTime     time.Time
	playbackStart time.Time
	isPlaying     bool
	speed         float64
	pauseCh       chan struct{}
	resumeCh      chan struct{}
	stopCh        chan struct{}
}

// ReplayConfig holds configuration for replay
type ReplayConfig struct {
	SourceFile   string        `json:"source_file"`
	LoopPlayback bool          `json:"loop_playback"`
	SpeedFactor  float64       `json:"speed_factor"` // 1.0 = real-time, 2.0 = 2x speed
	BufferSize   int           `json:"buffer_size"`  // Number of records to buffer
	StartOffset  time.Duration `json:"start_offset"` // Start time offset
	EndOffset    time.Duration `json:"end_offset"`   // End time offset (0 = to end)
}

// ReplayRecord represents a single replay record
type ReplayRecord struct {
	Timestamp  time.Time       `json:"timestamp"`
	SourceID   string          `json:"source_id"`
	RecordType string          `json:"record_type"`
	Data       json.RawMessage `json:"data"`
}

// DefaultReplayConfig returns default replay configuration
func DefaultReplayConfig() *ReplayConfig {
	return &ReplayConfig{
		LoopPlayback: false,
		SpeedFactor:  1.0,
		BufferSize:   1000,
		StartOffset:  0,
		EndOffset:    0,
	}
}

// NewReplayManager creates a new replay manager
func NewReplayManager(config *ReplayConfig) *ReplayManager {
	if config == nil {
		config = DefaultReplayConfig()
	}

	return &ReplayManager{
		config:   config,
		records:  make([]ReplayRecord, 0),
		speed:    config.SpeedFactor,
		pauseCh:  make(chan struct{}),
		resumeCh: make(chan struct{}),
		stopCh:   make(chan struct{}),
	}
}

// LoadFile loads a replay file
func (rm *ReplayManager) LoadFile(filename string) error {
	rm.playbackMu.Lock()
	defer rm.playbackMu.Unlock()

	// Close existing file
	if rm.file != nil {
		rm.file.Close()
	}

	// Open file
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	rm.file = file
	rm.config.SourceFile = filename

	// Detect gzip
	reader := bufio.NewReader(file)
	header, err := reader.Peek(2)
	if err == nil && len(header) >= 2 && header[0] == 0x1f && header[1] == 0x8b {
		// Gzip file
		file.Seek(0, 0)
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		rm.gzipReader = gzipReader
		rm.reader = bufio.NewReader(gzipReader)
	} else {
		// Plain file
		file.Seek(0, 0)
		rm.reader = reader
	}

	// Load records
	rm.records = make([]ReplayRecord, 0)
	decoder := json.NewDecoder(rm.reader)

	for {
		var record ReplayRecord
		err := decoder.Decode(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		rm.records = append(rm.records, record)
	}

	rm.currentIdx = 0

	// Apply start offset
	if rm.config.StartOffset > 0 && len(rm.records) > 0 {
		startTime := rm.records[0].Timestamp.Add(rm.config.StartOffset)
		for i, r := range rm.records {
			if !r.Timestamp.Before(startTime) {
				rm.currentIdx = i
				break
			}
		}
	}

	return nil
}

// Play starts playback
func (rm *ReplayManager) Play() <-chan ReplayRecord {
	output := make(chan ReplayRecord, rm.config.BufferSize)

	rm.playbackMu.Lock()
	rm.isPlaying = true
	rm.playbackStart = time.Now()
	rm.startTime = rm.records[rm.currentIdx].Timestamp
	rm.playbackMu.Unlock()

	go func() {
		defer close(output)

		for {
			select {
			case <-rm.stopCh:
				return
			case <-rm.pauseCh:
				// Wait for resume
				<-rm.resumeCh
			default:
				record, ok := rm.nextRecord()
				if !ok {
					if rm.config.LoopPlayback {
						rm.currentIdx = 0
						rm.playbackStart = time.Now()
						rm.startTime = rm.records[0].Timestamp
						continue
					}
					return
				}

				// Apply time scaling
				if rm.speed > 0 {
					recordTime := record.Timestamp.Sub(rm.startTime)
					scaledTime := time.Duration(float64(recordTime) / rm.speed)
					realTime := time.Since(rm.playbackStart)

					waitTime := scaledTime - realTime
					if waitTime > 0 {
						time.Sleep(waitTime)
					}
				}

				select {
				case output <- record:
				case <-rm.stopCh:
					return
				}
			}
		}
	}()

	return output
}

// nextRecord returns the next record
func (rm *ReplayManager) nextRecord() (ReplayRecord, bool) {
	rm.playbackMu.Lock()
	defer rm.playbackMu.Unlock()

	// Check end offset
	if rm.config.EndOffset > 0 && rm.currentIdx > 0 {
		elapsed := rm.records[rm.currentIdx].Timestamp.Sub(rm.records[0].Timestamp)
		if elapsed > rm.config.EndOffset {
			return ReplayRecord{}, false
		}
	}

	if rm.currentIdx >= len(rm.records) {
		return ReplayRecord{}, false
	}

	record := rm.records[rm.currentIdx]
	rm.currentIdx++

	return record, true
}

// Pause pauses playback
func (rm *ReplayManager) Pause() {
	rm.playbackMu.Lock()
	defer rm.playbackMu.Unlock()

	if rm.isPlaying {
		rm.isPlaying = false
		rm.pauseCh <- struct{}{}
	}
}

// Resume resumes playback
func (rm *ReplayManager) Resume() {
	rm.playbackMu.Lock()
	defer rm.playbackMu.Unlock()

	if !rm.isPlaying {
		rm.isPlaying = true
		rm.playbackStart = time.Now()
		rm.startTime = rm.records[rm.currentIdx].Timestamp
		rm.resumeCh <- struct{}{}
	}
}

// Stop stops playback
func (rm *ReplayManager) Stop() {
	rm.playbackMu.Lock()
	defer rm.playbackMu.Unlock()

	rm.isPlaying = false
	rm.currentIdx = 0
	close(rm.stopCh)
	rm.stopCh = make(chan struct{})
}

// SetSpeed sets playback speed
func (rm *ReplayManager) SetSpeed(speed float64) {
	rm.playbackMu.Lock()
	defer rm.playbackMu.Unlock()

	if speed < 0.1 {
		speed = 0.1
	}
	if speed > 100 {
		speed = 100
	}

	rm.speed = speed
	rm.config.SpeedFactor = speed

	// Recalculate playback start
	if rm.isPlaying && rm.currentIdx > 0 {
		rm.playbackStart = time.Now()
		rm.startTime = rm.records[rm.currentIdx].Timestamp
	}
}

// GetSpeed returns current playback speed
func (rm *ReplayManager) GetSpeed() float64 {
	rm.playbackMu.Lock()
	defer rm.playbackMu.Unlock()
	return rm.speed
}

// Seek seeks to a specific timestamp
func (rm *ReplayManager) Seek(target time.Time) error {
	rm.playbackMu.Lock()
	defer rm.playbackMu.Unlock()

	for i, r := range rm.records {
		if !r.Timestamp.Before(target) {
			rm.currentIdx = i
			if rm.isPlaying {
				rm.playbackStart = time.Now()
				rm.startTime = r.Timestamp
			}
			return nil
		}
	}

	return io.EOF
}

// SeekPercent seeks to a percentage of the recording
func (rm *ReplayManager) SeekPercent(percent float64) error {
	if len(rm.records) == 0 {
		return nil
	}

	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	startTime := rm.records[0].Timestamp
	endTime := rm.records[len(rm.records)-1].Timestamp
	totalDuration := endTime.Sub(startTime)

	target := startTime.Add(time.Duration(float64(totalDuration) * percent / 100))
	return rm.Seek(target)
}

// GetStats returns replay statistics
func (rm *ReplayManager) GetStats() ReplayStats {
	rm.playbackMu.Lock()
	defer rm.playbackMu.Unlock()

	if len(rm.records) == 0 {
		return ReplayStats{
			TotalRecords: 0,
			IsPlaying:    rm.isPlaying,
			Speed:        rm.speed,
		}
	}

	startTime := rm.records[0].Timestamp
	endTime := rm.records[len(rm.records)-1].Timestamp

	currentTime := startTime
	if rm.currentIdx < len(rm.records) {
		currentTime = rm.records[rm.currentIdx].Timestamp
	}

	progress := 0.0
	if endTime.After(startTime) {
		progress = float64(currentTime.Sub(startTime)) / float64(endTime.Sub(startTime)) * 100
	}

	return ReplayStats{
		TotalRecords: len(rm.records),
		CurrentIndex: rm.currentIdx,
		IsPlaying:    rm.isPlaying,
		Speed:        rm.speed,
		StartTime:    startTime,
		EndTime:      endTime,
		CurrentTime:  currentTime,
		Progress:     progress,
	}
}

// ReplayStats holds replay statistics
type ReplayStats struct {
	TotalRecords int       `json:"total_records"`
	CurrentIndex int       `json:"current_index"`
	IsPlaying    bool      `json:"is_playing"`
	Speed        float64   `json:"speed"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	CurrentTime  time.Time `json:"current_time"`
	Progress     float64   `json:"progress"` // 0-100
}

// Close closes the replay manager
func (rm *ReplayManager) Close() error {
	rm.playbackMu.Lock()
	defer rm.playbackMu.Unlock()

	rm.isPlaying = false

	if rm.gzipReader != nil {
		rm.gzipReader.Close()
	}

	if rm.file != nil {
		return rm.file.Close()
	}

	return nil
}

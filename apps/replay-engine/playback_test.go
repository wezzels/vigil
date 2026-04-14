// Package replay provides playback tests with time scaling
package replay

import (
	"testing"
	"time"
)

// TestPlaybackTimeScaling tests playback at different speeds
func TestPlaybackTimeScaling(t *testing.T) {
	tests := []struct {
		name     string
		realTime time.Duration
		playback time.Duration
		scale    float64
	}{
		{
			name:     "1x_speed",
			realTime: 10 * time.Second,
			playback: 10 * time.Second,
			scale:    1.0,
		},
		{
			name:     "2x_speed",
			realTime: 5 * time.Second,
			playback: 10 * time.Second,
			scale:    2.0,
		},
		{
			name:     "0.5x_speed",
			realTime: 20 * time.Second,
			playback: 10 * time.Second,
			scale:    0.5,
		},
		{
			name:     "10x_speed",
			realTime: 1 * time.Second,
			playback: 10 * time.Second,
			scale:    10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			playback := NewPlayback(tt.playback, tt.scale)

			start := time.Now()
			playback.Start()
			playback.Wait()
			elapsed := time.Since(start)

			// Allow 10% tolerance
			tolerance := float64(tt.realTime) * 0.1
			diff := float64(elapsed) - float64(tt.realTime)
			if diff < 0 {
				diff = -diff
			}

			if diff > tolerance {
				t.Errorf("Timing mismatch: expected ~%v, got %v", tt.realTime, elapsed)
			}
		})
	}
}

// TestPlaybackPauseResume tests pause/resume functionality
func TestPlaybackPauseResume(t *testing.T) {
	playback := NewPlayback(5*time.Second, 1.0)

	playback.Start()
	time.Sleep(1 * time.Second)
	playback.Pause()

	elapsed := playback.Elapsed()
	if elapsed < 900*time.Millisecond || elapsed > 1100*time.Millisecond {
		t.Errorf("Pause timing wrong: %v", elapsed)
	}

	// Wait while paused
	time.Sleep(500 * time.Millisecond)

	// Resume
	playback.Resume()
	playback.Wait()

	totalElapsed := playback.TotalElapsed()
	expected := 5*time.Second + 500*time.Millisecond // Original + paused duration
	tolerance := 500 * time.Millisecond

	diff := totalElapsed - expected
	if diff < 0 {
		diff = -diff
	}

	if diff > tolerance {
		t.Errorf("Total timing wrong: expected ~%v, got %v", expected, totalElapsed)
	}
}

// TestPlaybackSeek tests seeking to specific times
func TestPlaybackSeek(t *testing.T) {
	playback := NewPlayback(10*time.Second, 1.0)

	// Seek to middle
	playback.Seek(5 * time.Second)

	if playback.CurrentTime() != 5*time.Second {
		t.Errorf("Seek failed: expected %v, got %v", 5*time.Second, playback.CurrentTime())
	}

	// Seek to start
	playback.Seek(0)
	if playback.CurrentTime() != 0 {
		t.Errorf("Seek to start failed")
	}

	// Seek beyond end should clamp
	playback.Seek(20 * time.Second)
	if playback.CurrentTime() != 10*time.Second {
		t.Errorf("Seek beyond end should clamp")
	}
}

// TestPlaybackReverse tests reverse playback
func TestPlaybackReverse(t *testing.T) {
	playback := NewPlayback(10*time.Second, -1.0) // Negative scale = reverse

	playback.Start()
	playback.Wait()

	// Should end at start
	if playback.CurrentTime() != 0 {
		t.Errorf("Reverse playback should end at start")
	}
}

// Playback type

type Playback struct {
	duration    time.Duration
	scale       float64
	startTime   time.Time
	pausedTime  time.Duration
	isPaused    bool
	currentTime time.Duration
	done        chan struct{}
}

func NewPlayback(duration time.Duration, scale float64) *Playback {
	return &Playback{
		duration: duration,
		scale:    scale,
		done:     make(chan struct{}),
	}
}

func (p *Playback) Start() {
	p.startTime = time.Now()
	go func() {
		if p.scale > 0 {
			time.Sleep(time.Duration(float64(p.duration) / p.scale))
		} else {
			time.Sleep(p.duration)
		}
		p.currentTime = p.duration
		close(p.done)
	}()
}

func (p *Playback) Wait() {
	<-p.done
}

func (p *Playback) Pause() {
	p.isPaused = true
	p.pausedTime = time.Since(p.startTime)
}

func (p *Playback) Resume() {
	p.isPaused = false
}

func (p *Playback) Elapsed() time.Duration {
	return p.pausedTime
}

func (p *Playback) TotalElapsed() time.Duration {
	return p.pausedTime
}

func (p *Playback) Seek(t time.Duration) {
	if t > p.duration {
		t = p.duration
	}
	if t < 0 {
		t = 0
	}
	p.currentTime = t
}

func (p *Playback) CurrentTime() time.Duration {
	return p.currentTime
}
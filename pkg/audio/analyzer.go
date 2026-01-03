package audio

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
)

// AudioAnalysis contains the results of audio analysis
type AudioAnalysis struct {
	DurationSeconds   float64        `json:"duration_seconds"`
	BPM               float64        `json:"bpm"`
	Key               string         `json:"key"`
	Tempo             string         `json:"tempo"`
	BeatTimes         []float64      `json:"beat_times"`
	BeatCount         int            `json:"beat_count"`
	VocalSegments     []VocalSegment `json:"vocal_segments"`
	VocalSegmentCount int            `json:"vocal_segment_count"`
	SpectralCentroid  float64        `json:"spectral_centroid"`
	ZeroCrossingRate  float64        `json:"zero_crossing_rate"`
	SampleRate        int            `json:"sample_rate"`
	Success           bool           `json:"success"`
	Error             string         `json:"error,omitempty"`
	ErrorType         string         `json:"error_type,omitempty"`
}

// VocalSegment represents a detected vocal segment
type VocalSegment struct {
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Duration float64 `json:"duration"`
}

// AnalyzeAudio analyzes an audio file using the Python librosa script
func AnalyzeAudio(audioPath string) (*AudioAnalysis, error) {
	// Get absolute path to analyzer script
	// Assumes it's in pkg/audio/analyzer.py relative to project root
	scriptPath, err := filepath.Abs("pkg/audio/analyzer.py")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve analyzer script path: %w", err)
	}

	// Execute Python script
	cmd := exec.Command("python3", scriptPath, audioPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("analyzer script failed: %w, output: %s", err, string(output))
	}

	// Parse JSON output
	var analysis AudioAnalysis
	if err := json.Unmarshal(output, &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse analyzer output: %w, raw output: %s", err, string(output))
	}

	// Check if analysis was successful
	if !analysis.Success {
		return nil, fmt.Errorf("audio analysis failed: %s (%s)", analysis.Error, analysis.ErrorType)
	}

	return &analysis, nil
}

// GetVocalTimingInfo returns formatted vocal timing information
func (a *AudioAnalysis) GetVocalTimingInfo() string {
	if len(a.VocalSegments) == 0 {
		return "No vocal segments detected"
	}

	info := fmt.Sprintf("Found %d vocal segments:\n", a.VocalSegmentCount)
	for i, seg := range a.VocalSegments {
		info += fmt.Sprintf("  Segment %d: %.2fs - %.2fs (%.2fs duration)\n",
			i+1, seg.Start, seg.End, seg.Duration)
	}
	return info
}

// GetBeatInfo returns formatted beat information
func (a *AudioAnalysis) GetBeatInfo() string {
	return fmt.Sprintf("BPM: %.1f, Tempo: %s, Beats: %d over %.1fs",
		a.BPM, a.Tempo, a.BeatCount, a.DurationSeconds)
}

// Summary returns a human-readable summary of the analysis
func (a *AudioAnalysis) Summary() string {
	return fmt.Sprintf(
		"Duration: %.1fs | BPM: %.1f (%s) | Key: %s | Beats: %d | Vocal Segments: %d",
		a.DurationSeconds, a.BPM, a.Tempo, a.Key, a.BeatCount, a.VocalSegmentCount,
	)
}

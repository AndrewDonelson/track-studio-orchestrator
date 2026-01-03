package lyrics

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// WhisperWord represents a word with precise timing from Whisper
type WhisperWord struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// WhisperSegment represents a segment/line from Whisper output
type WhisperSegment struct {
	Text  string        `json:"text"`
	Start float64       `json:"start"`
	End   float64       `json:"end"`
	Words []WhisperWord `json:"words"`
}

// WhisperTranscription is the full Whisper output
type WhisperTranscription struct {
	Text     string           `json:"text"`
	Segments []WhisperSegment `json:"segments"`
}

// GetWordLevelTimings uses OpenAI Whisper to get precise word-level timings
// This requires whisper to be installed: pip install openai-whisper
func GetWordLevelTimings(audioPath string) ([]WhisperWord, error) {
	// Use whisper CLI with word-level timestamps
	// Format: whisper audio.mp3 --model base --output_format json --word_timestamps True

	outputDir := filepath.Dir(audioPath)
	baseName := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	jsonOutput := filepath.Join(outputDir, baseName+".json")

	cmd := exec.Command("whisper",
		audioPath,
		"--model", "base",
		"--output_format", "json",
		"--word_timestamps", "True",
		"--output_dir", outputDir,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("whisper failed: %w\nOutput: %s", err, string(output))
	}

	// Read the JSON output
	var transcription WhisperTranscription
	if err := readJSONFile(jsonOutput, &transcription); err != nil {
		return nil, fmt.Errorf("failed to read whisper output: %w", err)
	}

	// Extract all words with timestamps
	var allWords []WhisperWord
	for _, segment := range transcription.Segments {
		allWords = append(allWords, segment.Words...)
	}

	return allWords, nil
}

// AlignLyricsWithWhisper matches existing lyrics with Whisper word timings
// This provides the best of both worlds: your lyrics text + Whisper's timing
func AlignLyricsWithWhisper(lyrics []string, whisperWords []WhisperWord) ([]TimedLyric, error) {
	// Normalize lyrics to words
	var lyricsWords []string
	for _, line := range lyrics {
		words := strings.Fields(strings.ToLower(line))
		lyricsWords = append(lyricsWords, words...)
	}

	// Align lyrics words with whisper words using fuzzy matching
	// This handles slight differences in transcription
	aligned := make([]TimedLyric, 0, len(lyrics))
	whisperIdx := 0

	for _, line := range lyrics {
		lineWords := strings.Fields(strings.ToLower(line))
		if len(lineWords) == 0 {
			continue
		}

		startTime := 0.0
		endTime := 0.0
		matched := 0

		// Find matching words in whisper output
		for _, word := range lineWords {
			if whisperIdx >= len(whisperWords) {
				break
			}

			// Fuzzy match (handles punctuation differences)
			whisperWord := strings.ToLower(strings.Trim(whisperWords[whisperIdx].Word, ".,!?;:"))
			if strings.Contains(whisperWord, word) || strings.Contains(word, whisperWord) {
				if matched == 0 {
					startTime = whisperWords[whisperIdx].Start
				}
				endTime = whisperWords[whisperIdx].End
				matched++
				whisperIdx++
			} else {
				// Try next whisper word
				whisperIdx++
			}
		}

		if matched > 0 {
			aligned = append(aligned, TimedLyric{
				Text:      line,
				StartTime: startTime,
				EndTime:   endTime,
			})
		}
	}

	return aligned, nil
}

// TimedLyric represents a lyric line with start/end times
type TimedLyric struct {
	Text      string  `json:"text"`
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
}

func readJSONFile(path string, v interface{}) error {
	cmd := exec.Command("cat", path)
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	return json.Unmarshal(output, v)
}

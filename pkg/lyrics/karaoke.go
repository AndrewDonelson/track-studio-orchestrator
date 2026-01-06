package lyrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// KaraokeOptions holds customization settings for karaoke subtitles
type KaraokeOptions struct {
	FontFamily           string
	FontSize             int
	PrimaryColor         string
	PrimaryBorderColor   string
	HighlightColor       string
	HighlightBorderColor string
	Alignment            int
	MarginBottom         int
}

// DefaultKaraokeOptions returns default karaoke settings
func DefaultKaraokeOptions() *KaraokeOptions {
	return &KaraokeOptions{
		FontFamily:           "Arial",
		FontSize:             96,
		PrimaryColor:         "4169E1", // Royal Blue
		PrimaryBorderColor:   "FFFFFF", // White
		HighlightColor:       "FFD700", // Gold
		HighlightBorderColor: "FFFFFF", // White
		Alignment:            5,        // Center
		MarginBottom:         0,
	}
}

// KaraokeGenerator handles word-level timestamp generation and ASS subtitle creation
type KaraokeGenerator struct {
	PythonPath   string
	ScriptsDir   string
	WhisperModel string
	VenvPath     string
}

// WhisperResult contains the full transcription result
type WhisperResult struct {
	Segments []WhisperSegment `json:"segments"`
	Language string           `json:"language,omitempty"`
	Method   string           `json:"method,omitempty"` // whisperx or faster-whisper
	Text     string           `json:"text,omitempty"`   // Full transcription text
}

// NewKaraokeGenerator creates a new karaoke generator instance
// scriptsPath should be the full path to python-scripts directory (from config.PythonScripts)
func NewKaraokeGenerator(scriptsPath string) *KaraokeGenerator {
	// Detect venv path - check multiple locations
	venvPaths := []string{
		// If scriptsPath is in data directory, check nearby venv
		filepath.Join(filepath.Dir(scriptsPath), ".venv/bin/python"),
		// System python3 as fallback
		"python3",
	}

	venvPath := "python3" // fallback
	for _, path := range venvPaths {
		if path == "python3" {
			venvPath = path
			break
		}
		if _, err := os.Stat(path); err == nil {
			venvPath = path
			break
		}
	}

	return &KaraokeGenerator{
		PythonPath:   venvPath,
		ScriptsDir:   scriptsPath,
		WhisperModel: "base", // Use "base" for faster processing, "large-v3" for best quality
		VenvPath:     venvPath,
	}
}

// GenerateTimestamps generates word-level timestamps from vocals track
func (kg *KaraokeGenerator) GenerateTimestamps(vocalsPath string, outputJSON string) (*WhisperResult, error) {
	log.Printf("Generating word-level timestamps from: %s", vocalsPath)

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputJSON), 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Try API method first, fallback to local script
	result, err := kg.generateTimestampsViaAPI(vocalsPath, outputJSON)
	if err != nil {
		log.Printf("API method failed, falling back to local script: %v", err)
		result, err = kg.generateTimestampsViaScript(vocalsPath, outputJSON)
		if err != nil {
			return nil, fmt.Errorf("both API and local methods failed: %w", err)
		}
	}

	totalWords := 0
	for _, seg := range result.Segments {
		totalWords += len(seg.Words)
	}
	log.Printf("Generated %d segments with %d words total", len(result.Segments), totalWords)

	return result, nil
}

// generateTimestampsViaAPI calls the WhisperX API service
func (kg *KaraokeGenerator) generateTimestampsViaAPI(vocalsPath string, outputJSON string) (*WhisperResult, error) {
	// Open the audio file
	file, err := os.Open(vocalsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	// Create multipart form data
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Add file
	fw, err := writer.CreateFormFile("file", filepath.Base(vocalsPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err = io.Copy(fw, file); err != nil {
		return nil, fmt.Errorf("failed to copy file data: %w", err)
	}

	// Add other parameters
	writer.WriteField("language", "en")
	writer.WriteField("model", kg.WhisperModel)
	writer.WriteField("align_mode", "false")

	writer.Close()

	// Make HTTP request to WhisperX API
	apiURL := "http://192.168.1.76:8181/transcribe/sync"
	req, err := http.NewRequest("POST", apiURL, &b)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 10 * time.Minute} // Long timeout for processing
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	// Convert API response to WhisperResult format
	result, err := kg.convertAPIResponseToWhisperResult(apiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to convert API response: %w", err)
	}

	// Save to output file
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := os.WriteFile(outputJSON, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write output file: %w", err)
	}

	result.Method = "whisperx-api"
	return result, nil
}

// generateTimestampsViaScript uses the local Python script (fallback method)
func (kg *KaraokeGenerator) generateTimestampsViaScript(vocalsPath string, outputJSON string) (*WhisperResult, error) {
	cmd := exec.Command(
		kg.PythonPath,
		filepath.Join(kg.ScriptsDir, "generate_timestamps.py"),
		"--vocals", vocalsPath,
		"--output", outputJSON,
		"--model", kg.WhisperModel,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("timestamp generation failed: %w\nOutput: %s", err, string(output))
	}

	log.Printf("Faster-Whisper output:\n%s", string(output))

	// Load and return the result
	var result WhisperResult
	data, err := os.ReadFile(outputJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to read timestamps: %w", err)
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse timestamps: %w", err)
	}

	return &result, nil
}

// convertAPIResponseToWhisperResult converts WhisperX API response to WhisperResult format
func (kg *KaraokeGenerator) convertAPIResponseToWhisperResult(apiResponse map[string]interface{}) (*WhisperResult, error) {
	result := &WhisperResult{}

	// Extract JSON data
	jsonData, ok := apiResponse["json_data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing json_data in API response")
	}

	// Convert segments
	segmentsData, ok := jsonData["segments"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("missing segments in json_data")
	}

	for _, segData := range segmentsData {
		segMap, ok := segData.(map[string]interface{})
		if !ok {
			continue
		}

		segment := WhisperSegment{
			Start: segMap["start"].(float64),
			End:   segMap["end"].(float64),
			Text:  segMap["text"].(string),
		}

		// Convert words
		wordsData, ok := segMap["words"].([]interface{})
		if ok {
			for _, wordData := range wordsData {
				wordMap, ok := wordData.(map[string]interface{})
				if !ok {
					continue
				}

				word := WhisperWord{
					Start: wordMap["start"].(float64),
					End:   wordMap["end"].(float64),
					Word:  wordMap["word"].(string),
				}

				if score, ok := wordMap["score"].(float64); ok {
					word.Score = score
				} else if probability, ok := wordMap["probability"].(float64); ok {
					word.Score = probability
				}

				segment.Words = append(segment.Words, word)
			}
		}

		result.Segments = append(result.Segments, segment)
	}

	// Set transcription text
	if transcription, ok := apiResponse["transcription"].(string); ok {
		result.Text = transcription
	}

	return result, nil
}

// GenerateASSFile generates an ASS subtitle file with karaoke effects
// If lyricsKaraoke is provided, uses actual lyrics instead of Whisper transcription
func (kg *KaraokeGenerator) GenerateASSFile(timestampsJSON string, outputASS string, lyricsKaraoke string, options *KaraokeOptions) error {
	log.Printf("Generating ASS subtitles from: %s", timestampsJSON)

	if options == nil {
		options = DefaultKaraokeOptions()
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputASS), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Prepare command arguments
	cmdArgs := []string{
		filepath.Join(kg.ScriptsDir, "generate_karaoke_ass.py"),
		"--timestamps", timestampsJSON,
		"--output", outputASS,
		"--font-family", options.FontFamily,
		"--font-size", fmt.Sprintf("%d", options.FontSize),
		"--primary-color", options.PrimaryColor,
		"--primary-border-color", options.PrimaryBorderColor,
		"--highlight-color", options.HighlightColor,
		"--highlight-border-color", options.HighlightBorderColor,
		"--alignment", fmt.Sprintf("%d", options.Alignment),
		"--margin-bottom", fmt.Sprintf("%d", options.MarginBottom),
	}

	// If lyrics_karaoke is provided, write to temp file and pass to script
	if lyricsKaraoke != "" {
		lyricsFile := filepath.Join(filepath.Dir(outputASS), "lyrics_temp.txt")
		log.Printf("DEBUG: Writing lyrics_karaoke to temp file: %s (length: %d, first 100 chars: %s)",
			lyricsFile, len(lyricsKaraoke), lyricsKaraoke[:min(100, len(lyricsKaraoke))])
		if err := os.WriteFile(lyricsFile, []byte(lyricsKaraoke), 0644); err != nil {
			log.Printf("Warning: failed to write lyrics file: %v", err)
		} else {
			cmdArgs = append(cmdArgs, "--lyrics", lyricsFile)
			defer os.Remove(lyricsFile) // Clean up temp file
		}
	} else {
		log.Printf("DEBUG: No lyrics_karaoke provided, will use Whisper transcription")
	}

	cmd := exec.Command(kg.PythonPath, cmdArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ASS generation failed: %w\nOutput: %s", err, string(output))
	}

	log.Printf("ASS generation output:\n%s", string(output))
	return nil
}

// GenerateKaraokeSubtitles is the complete pipeline: vocals → timestamps → ASS
// If lyricsKaraoke is provided, uses actual lyrics for display instead of Whisper transcription
// Returns the ASS path and the whisper engine used (whisperx or faster-whisper)
func (kg *KaraokeGenerator) GenerateKaraokeSubtitles(vocalsPath string, songID int, workingDir string, lyricsKaraoke string, options *KaraokeOptions) (string, string, error) {
	// Define output paths
	timestampsJSON := filepath.Join(workingDir, fmt.Sprintf("song_%d_timestamps.json", songID))
	assPath := filepath.Join(workingDir, fmt.Sprintf("song_%d_karaoke.ass", songID))

	// Step 1: Generate timestamps (uses Whisper for timing only)
	result, err := kg.GenerateTimestamps(vocalsPath, timestampsJSON)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate timestamps: %w", err)
	}

	// Extract which engine was used
	whisperEngine := result.Method
	if whisperEngine == "" {
		whisperEngine = "faster-whisper" // default fallback
	}

	// Step 2: Generate ASS file (with actual lyrics if provided)
	err = kg.GenerateASSFile(timestampsJSON, assPath, lyricsKaraoke, options)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate ASS file: %w", err)
	}

	log.Printf("Successfully generated karaoke subtitles using %s: %s", whisperEngine, assPath)
	return assPath, whisperEngine, nil
}

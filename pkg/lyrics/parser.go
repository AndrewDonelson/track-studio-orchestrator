package lyrics

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Section represents a detected section in lyrics (verse, chorus, bridge, etc.)
type Section struct {
	Type      string   `json:"type"`       // "verse", "chorus", "bridge", "intro", "outro"
	Number    int      `json:"number"`     // Which occurrence (verse 1, verse 2, etc.)
	StartLine int      `json:"start_line"` // Line number where section starts
	EndLine   int      `json:"end_line"`   // Line number where section ends
	Lines     []string `json:"lines"`      // Actual lyrics lines
}

// TimedLine represents a single line with timing information
type TimedLine struct {
	Line      string  `json:"line"`       // The lyrics text
	StartTime float64 `json:"start_time"` // Start time in seconds
	EndTime   float64 `json:"end_time"`   // End time in seconds
	Duration  float64 `json:"duration"`   // Duration in seconds
}

// LyricsData contains parsed and structured lyrics with timing
type LyricsData struct {
	RawLyrics   string      `json:"raw_lyrics"`
	Sections    []Section   `json:"sections"`
	TimedLines  []TimedLine `json:"timed_lines"`
	TotalLines  int         `json:"total_lines"`
	HasSections bool        `json:"has_sections"`
}

// ParseLyrics parses raw lyrics text into structured sections
func ParseLyrics(rawLyrics string) (*LyricsData, error) {
	if strings.TrimSpace(rawLyrics) == "" {
		return nil, fmt.Errorf("empty lyrics")
	}

	lines := strings.Split(rawLyrics, "\n")
	var cleanLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	if len(cleanLines) == 0 {
		return nil, fmt.Errorf("no valid lyrics lines found")
	}

	data := &LyricsData{
		RawLyrics:  rawLyrics,
		TotalLines: len(cleanLines),
	}

	// Detect sections
	sections := detectSections(cleanLines)
	data.Sections = sections
	data.HasSections = len(sections) > 0

	return data, nil
}

// detectSections identifies verse, chorus, bridge sections in lyrics
func detectSections(lines []string) []Section {
	var sections []Section

	// Patterns for explicit section markers
	versePattern := regexp.MustCompile(`(?i)^\[?verse\s*(\d+)?\]?$`)
	chorusPattern := regexp.MustCompile(`(?i)^\[?chorus\]?$`)
	bridgePattern := regexp.MustCompile(`(?i)^\[?bridge\]?$`)
	introPattern := regexp.MustCompile(`(?i)^\[?intro\]?$`)
	outroPattern := regexp.MustCompile(`(?i)^\[?outro\]?$`)

	currentSection := Section{
		Type:      "verse",
		Number:    1,
		StartLine: 0,
		Lines:     []string{},
	}

	verseCount := 1
	chorusCount := 0
	inSection := false

	for i, line := range lines {
		// Check for explicit section markers
		if versePattern.MatchString(line) {
			if inSection {
				currentSection.EndLine = i - 1
				sections = append(sections, currentSection)
			}
			matches := versePattern.FindStringSubmatch(line)
			num := verseCount
			if len(matches) > 1 && matches[1] != "" {
				fmt.Sscanf(matches[1], "%d", &num)
			}
			currentSection = Section{
				Type:      "verse",
				Number:    num,
				StartLine: i + 1,
				Lines:     []string{},
			}
			verseCount++
			inSection = true
			continue
		}

		if chorusPattern.MatchString(line) {
			if inSection {
				currentSection.EndLine = i - 1
				sections = append(sections, currentSection)
			}
			chorusCount++
			currentSection = Section{
				Type:      "chorus",
				Number:    chorusCount,
				StartLine: i + 1,
				Lines:     []string{},
			}
			inSection = true
			continue
		}

		if bridgePattern.MatchString(line) {
			if inSection {
				currentSection.EndLine = i - 1
				sections = append(sections, currentSection)
			}
			currentSection = Section{
				Type:      "bridge",
				Number:    1,
				StartLine: i + 1,
				Lines:     []string{},
			}
			inSection = true
			continue
		}

		if introPattern.MatchString(line) {
			if inSection {
				currentSection.EndLine = i - 1
				sections = append(sections, currentSection)
			}
			currentSection = Section{
				Type:      "intro",
				Number:    1,
				StartLine: i + 1,
				Lines:     []string{},
			}
			inSection = true
			continue
		}

		if outroPattern.MatchString(line) {
			if inSection {
				currentSection.EndLine = i - 1
				sections = append(sections, currentSection)
			}
			currentSection = Section{
				Type:      "outro",
				Number:    1,
				StartLine: i + 1,
				Lines:     []string{},
			}
			inSection = true
			continue
		}

		// Add line to current section
		if !strings.HasPrefix(line, "[") {
			currentSection.Lines = append(currentSection.Lines, line)
		}
	}

	// Close final section
	if inSection && len(currentSection.Lines) > 0 {
		currentSection.EndLine = len(lines) - 1
		sections = append(sections, currentSection)
	}

	// If no explicit sections found, detect implicitly by repetition
	if len(sections) == 0 {
		sections = detectImplicitSections(lines)
	}

	return sections
}

// detectImplicitSections finds repeated sections (likely chorus) without explicit markers
func detectImplicitSections(lines []string) []Section {
	// Simple heuristic: group lines into 4-line chunks and look for repetition
	// More sophisticated algorithms could use edit distance

	var sections []Section
	chunkSize := 4
	chunks := make(map[string][]int) // chunk text -> line indices

	for i := 0; i < len(lines); i += chunkSize {
		end := i + chunkSize
		if end > len(lines) {
			end = len(lines)
		}

		chunk := strings.Join(lines[i:end], "\n")
		chunks[chunk] = append(chunks[chunk], i)
	}

	// Find most repeated chunk (likely chorus)
	var maxChunk string
	maxCount := 0
	for chunk, indices := range chunks {
		if len(indices) > maxCount {
			maxCount = len(indices)
			maxChunk = chunk
		}
	}

	// If we found a repeated section, mark it as chorus
	verseNum := 1
	chorusNum := 0

	for i := 0; i < len(lines); {
		end := i + chunkSize
		if end > len(lines) {
			end = len(lines)
		}

		chunk := strings.Join(lines[i:end], "\n")
		sectionType := "verse"
		sectionNum := verseNum

		if chunk == maxChunk && maxCount > 1 {
			sectionType = "chorus"
			chorusNum++
			sectionNum = chorusNum
		} else {
			verseNum++
		}

		sections = append(sections, Section{
			Type:      sectionType,
			Number:    sectionNum,
			StartLine: i,
			EndLine:   end - 1,
			Lines:     lines[i:end],
		})

		i = end
	}

	return sections
}

// AlignLyricsToBeats creates timed lyrics lines based on beat times
func AlignLyricsToBeats(lyrics string, beatTimes []float64, duration float64) ([]TimedLine, error) {
	lines := strings.Split(lyrics, "\n")
	var cleanLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "[") {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	if len(cleanLines) == 0 {
		return nil, fmt.Errorf("no valid lyrics lines")
	}

	if len(beatTimes) == 0 {
		// No beats, distribute evenly
		return distributeEvenly(cleanLines, duration), nil
	}

	var timedLines []TimedLine

	// Calculate how many beats per line
	beatsPerLine := float64(len(beatTimes)) / float64(len(cleanLines))
	if beatsPerLine < 1 {
		beatsPerLine = 1
	}

	beatIndex := 0
	for i, line := range cleanLines {
		if beatIndex >= len(beatTimes) {
			// Ran out of beats, use remaining duration
			startTime := beatTimes[len(beatTimes)-1]
			timedLines = append(timedLines, TimedLine{
				Line:      line,
				StartTime: startTime,
				EndTime:   duration,
				Duration:  duration - startTime,
			})
			continue
		}

		startTime := beatTimes[beatIndex]

		// Find end time (next line's start or end of beats)
		nextBeatIndex := beatIndex + int(beatsPerLine)
		if nextBeatIndex >= len(beatTimes) {
			nextBeatIndex = len(beatTimes) - 1
		}

		endTime := beatTimes[nextBeatIndex]
		if i == len(cleanLines)-1 {
			// Last line extends to end
			endTime = duration
		}

		timedLines = append(timedLines, TimedLine{
			Line:      line,
			StartTime: startTime,
			EndTime:   endTime,
			Duration:  endTime - startTime,
		})

		beatIndex = nextBeatIndex
	}

	return timedLines, nil
}

// distributeEvenly distributes lyrics lines evenly across duration
func distributeEvenly(lines []string, duration float64) []TimedLine {
	var timedLines []TimedLine
	timePerLine := duration / float64(len(lines))

	for i, line := range lines {
		startTime := float64(i) * timePerLine
		endTime := startTime + timePerLine

		timedLines = append(timedLines, TimedLine{
			Line:      line,
			StartTime: startTime,
			EndTime:   endTime,
			Duration:  timePerLine,
		})
	}

	return timedLines
}

// ToJSON converts LyricsData to JSON string
func (ld *LyricsData) ToJSON() (string, error) {
	data, err := json.Marshal(ld)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetSectionSummary returns a human-readable section summary
func (ld *LyricsData) GetSectionSummary() string {
	if !ld.HasSections || len(ld.Sections) == 0 {
		return "No sections detected"
	}

	summary := fmt.Sprintf("Found %d sections:\n", len(ld.Sections))
	for _, section := range ld.Sections {
		summary += fmt.Sprintf("  %s %d: %d lines (lines %d-%d)\n",
			strings.Title(section.Type), section.Number,
			len(section.Lines), section.StartLine, section.EndLine)
	}
	return summary
}

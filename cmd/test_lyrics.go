package main

import (
	"encoding/json"
	"fmt"

	"github.com/AndrewDonelson/track-studio-orchestrator/pkg/lyrics"
)

func main() {
	fmt.Println("🧪 Testing Lyrics Parser")
	fmt.Println("=========================\n")

	// Test lyrics with sections
	testLyrics := `In the land of love
Where hearts collide
We dance together
Side by side

Through the night we fly
Underneath the stars
In this land of love
Forever ours`

	// Parse lyrics
	fmt.Println("📝 Parsing lyrics...")
	lyricsData, err := lyrics.ParseLyrics(testLyrics)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Printf("✅ Parsed %d total lines\n", lyricsData.TotalLines)
	fmt.Printf("   Sections detected: %v\n\n", lyricsData.HasSections)

	// Show section summary
	fmt.Println("📋 Section Summary:")
	fmt.Println(lyricsData.GetSectionSummary())

	// Show sections as JSON
	fmt.Println("\n📄 Sections (JSON):")
	sectionsJSON, _ := json.MarshalIndent(lyricsData.Sections, "", "  ")
	fmt.Println(string(sectionsJSON))

	// Test timing alignment
	fmt.Println("\n⏱️  Testing timing alignment...")

	// Simulate beat times (every 0.5 seconds for 10 seconds)
	beatTimes := []float64{0, 0.5, 1.0, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0, 4.5, 5.0, 5.5, 6.0, 6.5, 7.0, 7.5, 8.0, 8.5, 9.0, 9.5, 10.0}
	duration := 10.0

	timedLines, err := lyrics.AlignLyricsToBeats(testLyrics, beatTimes, duration)
	if err != nil {
		fmt.Printf("❌ Error aligning: %v\n", err)
		return
	}

	fmt.Printf("✅ Aligned %d lines to beats\n\n", len(timedLines))

	// Show first 3 timed lines
	fmt.Println("📌 First 3 timed lines:")
	for i, line := range timedLines {
		if i >= 3 {
			break
		}
		fmt.Printf("   %.1fs-%.1fs: \"%s\"\n", line.StartTime, line.EndTime, line.Line)
	}

	// Show timed lines as JSON
	fmt.Println("\n📄 Timed Lines (JSON sample):")
	timedJSON, _ := json.MarshalIndent(timedLines[0:3], "", "  ")
	fmt.Println(string(timedJSON))

	fmt.Println("\n✅ All tests passed!")
}

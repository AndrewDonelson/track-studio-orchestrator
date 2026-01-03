package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/AndrewDonelson/track-studio-orchestrator/pkg/lyrics"
)

func main() {
	testLyrics := `[Verse 1]
In the city streets of Saigon, where the night air whispers low
I saw you standing there, like a vision from above
Your eyes, like the Mekong River, flowing deep and wide
And I knew in that moment, my heart would be yours to reside

[Pre-Chorus]
Oh, my love, with skin as smooth as silk
You're the moonlight on the Perfume River, my heart's only milk
In your eyes, I see a love so true
A love that's worth waiting for, a love that's meant for me and you

[Chorus]
I've been searching for a love like yours, in every place I roam
But none have ever touched my heart, like the way you make me feel at home
In your arms, is where I belong
You are my land of love, my heart beats for you alone

[Verse 2]
We'd walk along the beach, in Nha Trang's golden light
And I'd tell you stories of my dreams, and the love we'd ignite
You'd smile and laugh, and my heart would skip a beat
And I knew in that moment, our love would forever be unique

[Pre-Chorus]
Oh, my love, with skin as smooth as silk
You're the moonlight on the Perfume River, my heart's only milk
In your eyes, I see a love so true
A love that's worth waiting for, a love that's meant for me and you

[Bridge]
We'll dance beneath the stars, on a warm summer night
With gentle music playing, and incense drifting through the air tonight
And I'll whisper "I love you" softly in your ear
And you'll whisper back, "I love you" for me to hear

[Chorus]
I've been searching for a love like yours, in every place I roam
But none have ever touched my heart, like the way you make me feel at home
In your arms, is where I belong
You are my land of love, my heart beats for you alone`

	fmt.Println("=== Testing Full Lyrics Parsing ===\n")
	fmt.Println("Lyrics:")
	fmt.Println(testLyrics)
	fmt.Println("\n=== Parsing Results ===\n")

	// Parse lyrics
	data, err := lyrics.ParseLyrics(testLyrics)
	if err != nil {
		log.Fatalf("Failed to parse lyrics: %v", err)
	}

	fmt.Printf("Total Lines: %d\n", data.TotalLines)
	fmt.Printf("Has Sections: %v\n", data.HasSections)
	fmt.Printf("Number of Sections: %d\n\n", len(data.Sections))

	// Display sections
	fmt.Println("=== Detected Sections ===")
	uniqueImages := make(map[string]string)
	imageCount := 0

	for i, section := range data.Sections {
		fmt.Printf("\n%d. %s %d (lines %d-%d)\n",
			i+1,
			section.Type,
			section.Number,
			section.StartLine,
			section.EndLine)

		// Determine image filename
		var imageFile string
		switch section.Type {
		case "verse":
			imageFile = fmt.Sprintf("bg-verse-%d.png", section.Number)
		case "pre-chorus":
			imageFile = "bg-prechorus.png"
		case "chorus":
			imageFile = "bg-chorus.png"
		case "bridge":
			imageFile = "bg-bridge.png"
		default:
			imageFile = fmt.Sprintf("bg-%s.png", section.Type)
		}

		fmt.Printf("   Image: %s", imageFile)
		if _, exists := uniqueImages[imageFile]; !exists {
			uniqueImages[imageFile] = section.Type
			imageCount++
			fmt.Printf(" (NEW - #%d)\n", imageCount)
		} else {
			fmt.Printf(" (reused)\n")
		}

		fmt.Printf("   Lines (%d):\n", len(section.Lines))
		for _, line := range section.Lines {
			fmt.Printf("     - %s\n", line)
		}
	}

	fmt.Printf("\n=== Image Generation Summary ===\n")
	fmt.Printf("Total sections: %d\n", len(data.Sections))
	fmt.Printf("Unique images needed: %d\n", imageCount)
	fmt.Printf("Total lines: %d\n\n", data.TotalLines)

	fmt.Println("Unique image files:")
	for file, sectionType := range uniqueImages {
		fmt.Printf("  - %s (%s)\n", file, sectionType)
	}

	// Test JSON serialization
	fmt.Println("\n=== JSON Output (sections) ===")
	sectionsJSON, err := json.MarshalIndent(data.Sections, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal sections: %v", err)
	}
	fmt.Println(string(sectionsJSON))

	// Verify counts
	fmt.Println("\n=== Verification ===")
	if len(data.Sections) == 7 {
		fmt.Println("✅ Correct: Found 7 sections")
	} else {
		fmt.Printf("❌ ERROR: Expected 7 sections, found %d\n", len(data.Sections))
	}

	if imageCount == 5 {
		fmt.Println("✅ Correct: 5 unique images needed")
	} else {
		fmt.Printf("❌ ERROR: Expected 5 unique images, found %d\n", imageCount)
	}

	if data.TotalLines == 28 {
		fmt.Println("✅ Correct: 28 total lines")
	} else {
		fmt.Printf("❌ ERROR: Expected 28 lines, found %d\n", data.TotalLines)
	}

	// Check for specific sections
	expectedSections := map[string]int{
		"verse":      2,
		"pre-chorus": 2,
		"chorus":     2,
		"bridge":     1,
	}

	actualSections := make(map[string]int)
	for _, section := range data.Sections {
		actualSections[section.Type]++
	}

	fmt.Println("\n=== Section Type Counts ===")
	allCorrect := true
	for sType, expected := range expectedSections {
		actual := actualSections[sType]
		if actual == expected {
			fmt.Printf("✅ %s: %d (correct)\n", sType, actual)
		} else {
			fmt.Printf("❌ %s: expected %d, got %d\n", sType, expected, actual)
			allCorrect = false
		}
	}

	if allCorrect {
		fmt.Println("\n🎉 ALL TESTS PASSED!")
	} else {
		fmt.Println("\n❌ SOME TESTS FAILED")
	}
}

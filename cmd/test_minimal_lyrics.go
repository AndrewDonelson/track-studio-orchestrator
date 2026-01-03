package main

import (
	"fmt"
	"log"
	"strings"

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
You're the moonlight on the Perfume River, my heart's only milk`

	fmt.Println("=== Minimal Test ===\n")
	fmt.Println("Lyrics:")
	fmt.Println(testLyrics)
	fmt.Println("\n=== Lines ===")

	lines := strings.Split(testLyrics, "\n")
	for i, line := range lines {
		fmt.Printf("%d: %q\n", i, line)
	}

	fmt.Println("\n=== Parsing ===\n")

	data, err := lyrics.ParseLyrics(testLyrics)
	if err != nil {
		log.Fatalf("Failed to parse lyrics: %v", err)
	}

	fmt.Printf("Total Lines: %d\n", data.TotalLines)
	fmt.Printf("Has Sections: %v\n", data.HasSections)
	fmt.Printf("Number of Sections: %d\n\n", len(data.Sections))

	for i, section := range data.Sections {
		fmt.Printf("%d. %s %d\n", i+1, section.Type, section.Number)
		for j, line := range section.Lines {
			fmt.Printf("   %d: %s\n", j, line)
		}
	}
}

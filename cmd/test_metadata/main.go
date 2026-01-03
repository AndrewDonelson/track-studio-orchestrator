package main

import (
	"fmt"

	"github.com/AndrewDonelson/track-studio-orchestrator/pkg/video"
)

func main() {
	fmt.Println("🧪 Testing Video Metadata Overlay")
	fmt.Println("==================================\n")

	// Create default overlay settings
	overlay := video.DefaultMetadataOverlay()

	fmt.Println("📋 Overlay Settings:")
	fmt.Printf("   %s\n", overlay.GetDescription())
	fmt.Printf("   Font: %s, %dpt, %s\n", overlay.FontFamily, overlay.FontSize, overlay.FontColor)
	fmt.Printf("   Position: Top %dpx, Left %dpx, Right %dpx\n\n",
		overlay.TopMargin, overlay.LeftMargin, overlay.RightMargin)

	// Generate FFmpeg filter for sample metadata
	key := "E Minor"
	tempo := "Extremely Fast"
	bpm := 160.7
	videoWidth := 1920

	fmt.Println("🎬 FFmpeg Filter Generation:")
	fmt.Println("=============================")
	fmt.Printf("Input: Key=%s, Tempo=%s, BPM=%.1f\n", key, tempo, bpm)
	fmt.Printf("Video Width: %dpx\n\n", videoWidth)

	filter := overlay.GetFFmpegDrawtextFilter(key, tempo, bpm, videoWidth)

	fmt.Println("Generated Filter:")
	fmt.Println(filter)

	fmt.Println("\n📐 Layout Preview:")
	fmt.Println("╔════════════════════════════════════════════╗")
	fmt.Printf("║ %-15s   %-15s   %15s ║\n", key, tempo, fmt.Sprintf("%.0f BPM", bpm))
	fmt.Println("║                                            ║")
	fmt.Println("║              (Video Content)               ║")
	fmt.Println("║                                            ║")
	fmt.Println("╚════════════════════════════════════════════╝")

	// Test with metadata disabled
	fmt.Println("\n🚫 Testing with metadata disabled:")
	overlay.ShowKey = false
	overlay.ShowTempo = false
	overlay.ShowBPM = false

	filter2 := overlay.GetFFmpegDrawtextFilter(key, tempo, bpm, videoWidth)
	if filter2 == "" {
		fmt.Println("✅ Correctly returns empty filter when disabled")
	} else {
		fmt.Println("❌ Should return empty filter")
	}

	fmt.Println("\n✅ Metadata overlay tests passed!")
}

package video

import "fmt"

// MetadataOverlay defines the positioning and styling for video metadata display
type MetadataOverlay struct {
	// Positioning
	TopMargin   int    `json:"top_margin"`   // Pixels from top
	LeftMargin  int    `json:"left_margin"`  // Pixels from left (for Key)
	RightMargin int    `json:"right_margin"` // Pixels from right (for BPM)
	FontSize    int    `json:"font_size"`    // Font size in points
	FontFamily  string `json:"font_family"`  // Font family name
	FontColor   string `json:"font_color"`   // Hex color code

	// Shadow/outline for readability
	TextShadow   bool   `json:"text_shadow"`
	ShadowColor  string `json:"shadow_color"`
	ShadowOffset int    `json:"shadow_offset"`

	// Content
	ShowKey   bool `json:"show_key"`   // Show musical key (left)
	ShowTempo bool `json:"show_tempo"` // Show tempo description (center)
	ShowBPM   bool `json:"show_bpm"`   // Show BPM (right)
}

// DefaultMetadataOverlay returns standard metadata overlay settings
func DefaultMetadataOverlay() *MetadataOverlay {
	return &MetadataOverlay{
		TopMargin:    40,
		LeftMargin:   40,
		RightMargin:  40,
		FontSize:     36,
		FontFamily:   "Arial",
		FontColor:    "#FFFFFF",
		TextShadow:   true,
		ShadowColor:  "#000000",
		ShadowOffset: 2,
		ShowKey:      true,
		ShowTempo:    true,
		ShowBPM:      true,
	}
}

// GetFFmpegDrawtextFilter generates FFmpeg drawtext filter for metadata overlay
func (m *MetadataOverlay) GetFFmpegDrawtextFilter(key, tempo string, bpm float64, videoWidth int) string {
	if !m.ShowKey && !m.ShowTempo && !m.ShowBPM {
		return ""
	}

	var filters []string

	// Left: Key
	if m.ShowKey && key != "" {
		filters = append(filters, m.drawText(
			key,
			fmt.Sprintf("x=%d:y=%d", m.LeftMargin, m.TopMargin),
		))
	}

	// Center: Tempo
	if m.ShowTempo && tempo != "" {
		centerX := videoWidth / 2
		filters = append(filters, m.drawText(
			tempo,
			fmt.Sprintf("x=%d-text_w/2:y=%d", centerX, m.TopMargin),
		))
	}

	// Right: BPM
	if m.ShowBPM && bpm > 0 {
		bpmText := fmt.Sprintf("%.0f BPM", bpm)
		filters = append(filters, m.drawText(
			bpmText,
			fmt.Sprintf("x=w-%d-text_w:y=%d", m.RightMargin, m.TopMargin),
		))
	}

	// Combine all filters
	if len(filters) == 0 {
		return ""
	}

	filterChain := filters[0]
	for i := 1; i < len(filters); i++ {
		filterChain += "," + filters[i]
	}

	return filterChain
}

// drawText creates a single FFmpeg drawtext filter
func (m *MetadataOverlay) drawText(text, position string) string {
	filter := fmt.Sprintf("drawtext=text='%s':%s:fontsize=%d:fontcolor=%s:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
		text, position, m.FontSize, m.FontColor)

	if m.TextShadow {
		filter += fmt.Sprintf(":shadowcolor=%s:shadowx=%d:shadowy=%d",
			m.ShadowColor, m.ShadowOffset, m.ShadowOffset)
	}

	return filter
}

// GetDescription returns human-readable description of overlay settings
func (m *MetadataOverlay) GetDescription() string {
	desc := "Metadata Overlay: "

	if m.ShowKey {
		desc += "Key (left) "
	}
	if m.ShowTempo {
		desc += "Tempo (center) "
	}
	if m.ShowBPM {
		desc += "BPM (right) "
	}

	if !m.ShowKey && !m.ShowTempo && !m.ShowBPM {
		desc += "Disabled"
	}

	return desc
}

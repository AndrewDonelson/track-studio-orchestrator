#!/usr/bin/env python3
"""
Generate ASS subtitle file with karaoke effects from timestamps
"""
import json
import argparse
import sys
from dataclasses import dataclass

@dataclass
class KaraokeConfig:
    font_size: int = 48
    primary_color: str = "4169E1"  # Royal Blue (BGR format)
    highlight_color: str = "FFD700"  # Gold (BGR format)
    outline_width: int = 3
    outline_color: str = "FFFFFF"  # White (BGR format)
    shadow_depth: int = 2
    margin_bottom: int = 0  # No margin for center positioning
    alignment: int = 5  # Center (horizontally and vertically)

def hex_to_ass_color(hex_color):
    """Convert hex color (RGB) to ASS color format (&HAABBGGRR&)"""
    # Remove # if present
    hex_color = hex_color.lstrip('#')
    
    # ASS uses BGR format with alpha channel
    r, g, b = hex_color[0:2], hex_color[2:4], hex_color[4:6]
    return f"&H00{b}{g}{r}&"

def create_karaoke_ass(timestamps_json, output_ass, config=KaraokeConfig()):
    """
    Generate ASS subtitle file with karaoke effects
    """
    # Load timestamps
    with open(timestamps_json, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    # Create ASS document header
    ass_content = f"""[Script Info]
Title: Karaoke Subtitles
ScriptType: v4.00+
WrapStyle: 0
PlayResX: 1920
PlayResY: 1080
ScaledBorderAndShadow: yes

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Karaoke,Arial,{config.font_size},{hex_to_ass_color(config.primary_color)},{hex_to_ass_color(config.highlight_color)},{hex_to_ass_color(config.outline_color)},&H80000000&,-1,0,0,0,100,100,0,0,1,{config.outline_width},{config.shadow_depth},{config.alignment},50,50,{config.margin_bottom},1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
"""
    
    # Process each segment
    events = []
    for segment in data['segments']:
        if 'words' not in segment:
            continue
            
        words = segment['words']
        segment_start = segment['start']
        segment_end = segment['end']
        
        # Build karaoke line with \k tags
        karaoke_text = ""
        for word in words:
            # Calculate centiseconds for \k tag
            duration_cs = int((word['end'] - word['start']) * 100)
            karaoke_text += f"{{\\k{duration_cs}}}{word['word']} "
        
        # Format timestamp for ASS (H:MM:SS.CC)
        start_time = format_ass_time(segment_start)
        end_time = format_ass_time(segment_end)
        
        # Create dialogue line
        event = f"Dialogue: 0,{start_time},{end_time},Karaoke,,0,0,0,,{karaoke_text.strip()}"
        events.append(event)
    
    ass_content += "\n".join(events)
    
    # Save ASS file
    with open(output_ass, 'w', encoding='utf-8') as f:
        f.write(ass_content)
    
    print(f"✓ Karaoke ASS saved to {output_ass}")
    print(f"✓ Generated {len(events)} subtitle events")

def format_ass_time(seconds):
    """Convert seconds to ASS timestamp format (H:MM:SS.CC)"""
    hours = int(seconds // 3600)
    minutes = int((seconds % 3600) // 60)
    secs = int(seconds % 60)
    centisecs = int((seconds % 1) * 100)
    return f"{hours}:{minutes:02d}:{secs:02d}.{centisecs:02d}"

def main():
    parser = argparse.ArgumentParser(description='Generate ASS karaoke subtitles from timestamps')
    parser.add_argument('--timestamps', required=True, help='Input timestamps JSON file')
    parser.add_argument('--output', required=True, help='Output ASS file')
    parser.add_argument('--font-size', type=int, default=48, help='Font size (default: 48)')
    
    args = parser.parse_args()
    
    try:
        config = KaraokeConfig(font_size=args.font_size)
        create_karaoke_ass(args.timestamps, args.output, config)
        sys.exit(0)
    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()

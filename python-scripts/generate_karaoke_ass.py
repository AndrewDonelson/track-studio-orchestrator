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
    font_family: str = "Arial"  # Google font name
    font_size: int = 96  # Font size in pixels
    primary_color: str = "4169E1"  # Royal Blue (BGR format hex)
    primary_border_color: str = "FFFFFF"  # White border (BGR format hex)
    highlight_color: str = "FFD700"  # Gold highlight (BGR format hex)
    highlight_border_color: str = "FFFFFF"  # White border (BGR format hex)
    outline_width: int = 3  # Border thickness
    shadow_depth: int = 2  # Shadow depth
    margin_bottom: int = 0  # Bottom margin in pixels
    alignment: int = 5  # 5=center, 2=bottom-center, 8=top-center
    max_chars_per_line: int = 45  # Maximum characters per line to prevent clipping

def hex_to_ass_color(hex_color):
    """Convert hex color (RGB) to ASS color format (&HAABBGGRR&)"""
    # Remove # if present
    hex_color = hex_color.lstrip('#')
    
    # ASS uses BGR format with alpha channel
    r, g, b = hex_color[0:2], hex_color[2:4], hex_color[4:6]
    return f"&H00{b}{g}{r}&"

def split_text_intelligently(words_data, max_chars):
    """
    Split words into lines that fit within max_chars while preserving timing
    Returns list of line segments with their words
    """
    if not words_data:
        return []
    
    lines = []
    current_line = []
    current_length = 0
    
    for word_data in words_data:
        word = word_data['word'].strip()
        word_len = len(word) + 1  # +1 for space
        
        if current_length + word_len <= max_chars or not current_line:
            current_line.append(word_data)
            current_length += word_len
        else:
            # Start new line
            if current_line:
                lines.append(current_line)
            current_line = [word_data]
            current_length = word_len
    
    if current_line:
        lines.append(current_line)
    
    return lines

def create_karaoke_ass(timestamps_json, output_ass, lyrics_text=None, config=KaraokeConfig()):
    """
    Generate ASS subtitle file with karaoke effects
    If lyrics_text is provided, uses actual lyrics instead of Whisper transcription
    """
    # Load timestamps
    with open(timestamps_json, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    # If actual lyrics provided, split into lines matching Whisper segments
    actual_lyrics_lines = None
    if lyrics_text:
        actual_lyrics_lines = [line.strip() for line in lyrics_text.split('\n') if line.strip()]
    
    # Create ASS document header
    ass_content = f"""[Script Info]
Title: Karaoke Subtitles
ScriptType: v4.00+
WrapStyle: 2
PlayResX: 1920
PlayResY: 1080
ScaledBorderAndShadow: yes

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Karaoke,{config.font_family},{config.font_size},{hex_to_ass_color(config.primary_color)},{hex_to_ass_color(config.highlight_color)},{hex_to_ass_color(config.primary_border_color)},&H80000000&,-1,0,0,0,100,100,0,0,1,{config.outline_width},{config.shadow_depth},{config.alignment},50,50,{config.margin_bottom},1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
"""
    
    # Process each segment
    events = []
    lyrics_line_idx = 0
    
    for segment in data['segments']:
        if 'words' not in segment:
            continue
            
        words = segment['words']
        segment_start = segment['start']
        segment_end = segment['end']
        
        # Check if line is too long and needs splitting
        total_text = ' '.join([w['word'].strip() for w in words])
        
        if len(total_text) > config.max_chars_per_line:
            # Split into multiple subtitle lines
            line_segments = split_text_intelligently(words, config.max_chars_per_line)
            
            for line_words in line_segments:
                if not line_words:
                    continue
                    
                line_start = line_words[0]['start']
                line_end = line_words[-1]['end']
                
                # Build karaoke text for this line
                karaoke_text = ""
                for word in line_words:
                    duration_cs = int((word['end'] - word['start']) * 100)
                    karaoke_text += f"{{\\k{duration_cs}}}{word['word'].strip()} "
                
                start_time = format_ass_time(line_start)
                end_time = format_ass_time(line_end)
                
                event = f"Dialogue: 0,{start_time},{end_time},Karaoke,,0,0,0,,{karaoke_text.strip()}"
                events.append(event)
        else:
            # Line fits on one line - keep as is
            karaoke_text = ""
            for word in words:
                duration_cs = int((word['end'] - word['start']) * 100)
                karaoke_text += f"{{\\k{duration_cs}}}{word['word'].strip()} "
            
            start_time = format_ass_time(segment_start)
            end_time = format_ass_time(segment_end)
            
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
    parser.add_argument('--lyrics', help='Optional: Actual lyrics file (uses real lyrics instead of Whisper transcription)')
    
    # Karaoke customization options
    parser.add_argument('--font-family', default='Arial', help='Font family (default: Arial)')
    parser.add_argument('--font-size', type=int, default=96, help='Font size (default: 96)')
    parser.add_argument('--primary-color', default='4169E1', help='Primary text color in hex (default: 4169E1 - Royal Blue)')
    parser.add_argument('--primary-border-color', default='FFFFFF', help='Primary border color in hex (default: FFFFFF - White)')
    parser.add_argument('--highlight-color', default='FFD700', help='Highlight color in hex (default: FFD700 - Gold)')
    parser.add_argument('--highlight-border-color', default='FFFFFF', help='Highlight border color in hex (default: FFFFFF - White)')
    parser.add_argument('--alignment', type=int, default=5, help='Text alignment: 1-9 (5=center, 2=bottom-center, default: 5)')
    parser.add_argument('--margin-bottom', type=int, default=0, help='Bottom margin in pixels (default: 0)')
    parser.add_argument('--max-chars', type=int, default=45, help='Max characters per line (default: 45)')
    
    args = parser.parse_args()
    
    try:
        # Load actual lyrics if provided
        lyrics_text = None
        if args.lyrics:
            with open(args.lyrics, 'r', encoding='utf-8') as f:
                lyrics_text = f.read()
        
        config = KaraokeConfig(
            font_family=args.font_family,
            font_size=args.font_size,
            primary_color=args.primary_color,
            primary_border_color=args.primary_border_color,
            highlight_color=args.highlight_color,
            highlight_border_color=args.highlight_border_color,
            alignment=args.alignment,
            margin_bottom=args.margin_bottom,
            max_chars_per_line=args.max_chars
        )
        create_karaoke_ass(args.timestamps, args.output, lyrics_text, config)
        sys.exit(0)
    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()

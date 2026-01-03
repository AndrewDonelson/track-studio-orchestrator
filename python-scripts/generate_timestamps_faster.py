#!/usr/bin/env python3
"""
Generate word-level timestamps using faster-whisper (simpler, fewer dependencies)
"""
from faster_whisper import WhisperModel
import json
import argparse
import sys

def generate_word_timestamps(vocals_path, output_json, model_size="base"):
    """
    Generate word-level timestamps using faster-whisper
    """
    print(f"Loading Faster-Whisper model: {model_size}")
    
    # Use CPU to avoid CUDA/CuDNN dependency issues
    # CPU is fast enough for real-time transcription
    model = WhisperModel(model_size, device="cpu", compute_type="int8")
    print("Using CPU device")
    
    print(f"Transcribing {vocals_path}...")
    
    # Transcribe with word-level timestamps
    segments, info = model.transcribe(
        vocals_path,
        word_timestamps=True,
        language="en",
        vad_filter=True,
        vad_parameters=dict(min_silence_duration_ms=500)
    )
    
    # Convert to JSON format compatible with ASS generator
    result = {"segments": [], "language": info.language}
    
    for segment in segments:
        seg_data = {
            "start": segment.start,
            "end": segment.end,
            "text": segment.text.strip(),
            "words": []
        }
        
        if segment.words:
            for word in segment.words:
                seg_data["words"].append({
                    "word": word.word.strip(),
                    "start": word.start,
                    "end": word.end,
                    "score": word.probability
                })
        
        result["segments"].append(seg_data)
    
    # Save results
    with open(output_json, 'w', encoding='utf-8') as f:
        json.dump(result, f, indent=2, ensure_ascii=False)
    
    total_words = sum(len(seg["words"]) for seg in result["segments"])
    print(f"✓ Timestamps saved to {output_json}")
    print(f"✓ Transcribed {len(result['segments'])} segments with {total_words} words")
    
    return result

def main():
    parser = argparse.ArgumentParser(description='Generate word-level timestamps using faster-whisper')
    parser.add_argument('--vocals', required=True, help='Path to vocals.wav')
    parser.add_argument('--output', required=True, help='Output JSON file')
    parser.add_argument('--model', default='base', help='Whisper model size (tiny, base, small, medium, large-v3)')
    
    args = parser.parse_args()
    
    try:
        generate_word_timestamps(args.vocals, args.output, args.model)
        sys.exit(0)
    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == "__main__":
    main()

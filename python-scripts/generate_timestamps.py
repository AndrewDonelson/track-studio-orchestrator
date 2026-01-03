#!/usr/bin/env python3
"""
Generate word-level timestamps from vocals track using WhisperX
"""
import torch
# Monkey patch torch.load to use weights_only=False for PyTorch 2.6 compatibility
_original_load = torch.load
def patched_load(*args, **kwargs):
    kwargs.setdefault('weights_only', False)
    return _original_load(*args, **kwargs)
torch.load = patched_load

import whisperx
import json
import gc
import argparse
import sys

def generate_word_timestamps(vocals_path, output_json, model_name="large-v3"):
    """
    Generate word-level timestamps from vocals track
    
    Args:
        vocals_path: Path to vocals.wav
        output_json: Where to save timestamps
        model_name: WhisperX model to use
    """
    device = "cuda" if torch.cuda.is_available() else "cpu"
    batch_size = 16  # Adjust based on GPU memory
    compute_type = "float16" if device == "cuda" else "int8"
    
    print(f"Using device: {device}")
    print(f"Loading WhisperX model: {model_name}")
    
    # 1. Load model
    model = whisperx.load_model(
        model_name, 
        device, 
        compute_type=compute_type,
        language="en"
    )
    
    # 2. Transcribe with timestamps
    print(f"Transcribing {vocals_path}...")
    audio = whisperx.load_audio(vocals_path)
    result = model.transcribe(audio, batch_size=batch_size)
    
    # 3. Align whisper output with forced alignment
    print("Aligning timestamps...")
    model_a, metadata = whisperx.load_align_model(
        language_code=result["language"], 
        device=device
    )
    result = whisperx.align(
        result["segments"], 
        model_a, 
        metadata, 
        audio, 
        device,
        return_char_alignments=False
    )
    
    # 4. Save results
    with open(output_json, 'w', encoding='utf-8') as f:
        json.dump(result, f, indent=2, ensure_ascii=False)
    
    # 5. Clean up GPU memory
    gc.collect()
    if device == "cuda":
        torch.cuda.empty_cache()
    
    # Print statistics
    total_words = sum(len(seg.get('words', [])) for seg in result['segments'])
    print(f"✓ Timestamps saved to {output_json}")
    print(f"✓ Transcribed {len(result['segments'])} segments with {total_words} words")
    
    return result

def main():
    parser = argparse.ArgumentParser(description='Generate word-level timestamps from vocals')
    parser.add_argument('--vocals', required=True, help='Path to vocals.wav')
    parser.add_argument('--output', required=True, help='Output JSON file')
    parser.add_argument('--model', default='large-v3', help='WhisperX model (default: large-v3)')
    
    args = parser.parse_args()
    
    try:
        generate_word_timestamps(args.vocals, args.output, args.model)
        sys.exit(0)
    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()

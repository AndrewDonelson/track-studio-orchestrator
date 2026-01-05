#!/usr/bin/env python3
"""
Generate word-level timestamps with automatic fallback:
1. Try WhisperX on GPU (best quality, fastest)
2. Fallback to Faster-Whisper on CPU (good quality, reliable)
"""
import json
import argparse
import sys

def try_whisperx(vocals_path, output_json, model_name="large-v3"):
    """
    Try WhisperX on GPU first
    """
    try:
        import torch
        # Monkey patch torch.load for PyTorch 2.6 compatibility
        _original_load = torch.load
        def patched_load(*args, **kwargs):
            kwargs.setdefault('weights_only', False)
            return _original_load(*args, **kwargs)
        torch.load = patched_load

        import whisperx
        import gc
        
        device = "cuda" if torch.cuda.is_available() else None
        
        if device is None:
            print("CUDA not available for WhisperX, falling back to Faster-Whisper")
            return False
        
        batch_size = 16
        compute_type = "float16"
        
        print(f"Using WhisperX on device: {device}")
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
        
        # 3. Align whisper output
        print("Aligning timestamps...")
        model_a, metadata = whisperx.load_align_model(
            language_code=result.get("language", "en"),
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
        
        # Cleanup
        gc.collect()
        torch.cuda.empty_cache()
        del model
        del model_a
        
        # 4. Save result
        output_data = {"segments": result["segments"], "language": "en", "method": "whisperx"}
        
        with open(output_json, 'w', encoding='utf-8') as f:
            json.dump(output_data, f, indent=2, ensure_ascii=False)
        
        total_words = sum(len(seg.get("words", [])) for seg in output_data["segments"])
        print(f"✓ WhisperX: Saved to {output_json}")
        print(f"✓ Transcribed {len(output_data['segments'])} segments with {total_words} words")
        
        return True
        
    except Exception as e:
        print(f"WhisperX failed: {e}")
        print("Falling back to Faster-Whisper on CPU...")
        return False

def use_faster_whisper(vocals_path, output_json, model_size="base"):
    """
    Fallback to Faster-Whisper on CPU
    """
    from faster_whisper import WhisperModel
    
    print(f"Loading Faster-Whisper model: {model_size} on CPU")
    model = WhisperModel(model_size, device="cpu", compute_type="int8")
    
    print(f"Transcribing {vocals_path}...")
    segments, info = model.transcribe(
        vocals_path,
        word_timestamps=True,
        language="en",
        vad_filter=True,
        vad_parameters=dict(min_silence_duration_ms=500)
    )
    
    # Convert to JSON format
    result = {"segments": [], "language": info.language, "method": "faster-whisper"}
    
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
    print(f"✓ Faster-Whisper: Saved to {output_json}")
    print(f"✓ Transcribed {len(result['segments'])} segments with {total_words} words")

def generate_word_timestamps(vocals_path, output_json, model_name="large-v3", force_cpu=False):
    """
    Generate word-level timestamps with automatic fallback
    
    Args:
        vocals_path: Path to vocals.wav
        output_json: Where to save timestamps  
        model_name: Model size (for both WhisperX and Faster-Whisper)
        force_cpu: Skip WhisperX and use Faster-Whisper directly
    """
    # Try WhisperX first unless forcing CPU
    if not force_cpu:
        if try_whisperx(vocals_path, output_json, model_name):
            return
    
    # Fallback to Faster-Whisper
    # Map WhisperX model names to Faster-Whisper sizes
    model_map = {
        "large-v3": "large-v3",
        "large-v2": "large-v2", 
        "large": "large-v2",
        "medium": "medium",
        "small": "small",
        "base": "base",
        "tiny": "tiny"
    }
    faster_model = model_map.get(model_name, "base")
    use_faster_whisper(vocals_path, output_json, faster_model)

def main():
    parser = argparse.ArgumentParser(description='Generate word-level timestamps with GPU/CPU fallback')
    parser.add_argument('--vocals', required=True, help='Path to vocals.wav')
    parser.add_argument('--output', required=True, help='Output JSON file')
    parser.add_argument('--model', default='large-v3', help='Model size (default: large-v3)')
    parser.add_argument('--force-cpu', action='store_true', help='Force CPU mode (skip WhisperX)')
    
    args = parser.parse_args()
    
    try:
        generate_word_timestamps(args.vocals, args.output, args.model, args.force_cpu)
        sys.exit(0)
    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == "__main__":
    main()

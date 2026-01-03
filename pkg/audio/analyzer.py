#!/usr/bin/env python3
"""
Audio Analysis Service for Track Studio
Analyzes audio files for BPM, key, duration, and vocal timing using librosa
"""
import sys
import json
import warnings
import librosa
import numpy as np
from typing import Dict, List, Tuple

# Suppress warnings
warnings.filterwarnings('ignore')


def estimate_key(chroma: np.ndarray) -> str:
    """Estimate musical key from chroma features"""
    # Key profiles (Krumhansl-Schmuckler key-finding algorithm)
    major_profile = np.array([6.35, 2.23, 3.48, 2.33, 4.38, 4.09, 2.52, 5.19, 2.39, 3.66, 2.29, 2.88])
    minor_profile = np.array([6.33, 2.68, 3.52, 5.38, 2.60, 3.53, 2.54, 4.75, 3.98, 2.69, 3.34, 3.17])
    
    key_names = ['C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B']
    
    # Average chroma over time
    chroma_mean = np.mean(chroma, axis=1)
    
    # Correlate with key profiles
    major_correlations = []
    minor_correlations = []
    
    for i in range(12):
        major_correlations.append(np.corrcoef(np.roll(major_profile, i), chroma_mean)[0, 1])
        minor_correlations.append(np.corrcoef(np.roll(minor_profile, i), chroma_mean)[0, 1])
    
    # Find best match
    max_major = max(major_correlations)
    max_minor = max(minor_correlations)
    
    if max_major > max_minor:
        key_idx = major_correlations.index(max_major)
        return f"{key_names[key_idx]} Major"
    else:
        key_idx = minor_correlations.index(max_minor)
        return f"{key_names[key_idx]} Minor"


def get_tempo_description(bpm: float) -> str:
    """Convert BPM to tempo description"""
    if bpm < 60:
        return "Very Slow"
    elif bpm < 80:
        return "Slow"
    elif bpm < 100:
        return "Moderate"
    elif bpm < 120:
        return "Medium Fast"
    elif bpm < 140:
        return "Fast"
    elif bpm < 160:
        return "Very Fast"
    else:
        return "Extremely Fast"


def detect_vocal_segments(y: np.ndarray, sr: int, beat_times: np.ndarray) -> List[Dict]:
    """
    Detect vocal segments using RMS energy analysis
    Returns timing information for vocal activity
    """
    # Compute RMS energy in short windows
    hop_length = 512
    frame_length = 2048
    rms = librosa.feature.rms(y=y, frame_length=frame_length, hop_length=hop_length)[0]
    
    # Convert frames to time
    times = librosa.frames_to_time(np.arange(len(rms)), sr=sr, hop_length=hop_length)
    
    # Threshold for vocal activity (adaptive based on mean energy)
    threshold = np.mean(rms) * 0.5
    
    # Find segments where energy exceeds threshold
    is_vocal = rms > threshold
    
    # Group consecutive frames into segments
    segments = []
    in_segment = False
    start_time = 0
    
    for i, (time, vocal) in enumerate(zip(times, is_vocal)):
        if vocal and not in_segment:
            # Start new segment
            start_time = time
            in_segment = True
        elif not vocal and in_segment:
            # End segment (if long enough)
            if time - start_time > 0.5:  # Minimum 0.5 second segment
                segments.append({
                    'start': float(start_time),
                    'end': float(time),
                    'duration': float(time - start_time)
                })
            in_segment = False
    
    # Close final segment if still open
    if in_segment:
        segments.append({
            'start': float(start_time),
            'end': float(times[-1]),
            'duration': float(times[-1] - start_time)
        })
    
    return segments


def analyze_audio(file_path: str) -> Dict:
    """
    Analyze audio file for BPM, key, duration, and vocal timing
    
    Args:
        file_path: Path to audio file (WAV, MP3, FLAC, etc.)
        
    Returns:
        Dictionary with analysis results
    """
    try:
        # Load audio
        y, sr = librosa.load(file_path, sr=None)
        
        # Duration
        duration = librosa.get_duration(y=y, sr=sr)
        
        # Tempo (BPM) and beat tracking
        tempo, beat_frames = librosa.beat.beat_track(y=y, sr=sr)
        beat_times = librosa.frames_to_time(beat_frames, sr=sr)
        
        # Key detection using chroma features
        chroma = librosa.feature.chroma_cqt(y=y, sr=sr)
        key = estimate_key(chroma)
        
        # Detect vocal segments
        vocal_segments = detect_vocal_segments(y, sr, beat_times)
        
        # Calculate additional metrics
        spectral_centroid = float(np.mean(librosa.feature.spectral_centroid(y=y, sr=sr)))
        zero_crossing_rate = float(np.mean(librosa.feature.zero_crossing_rate(y)))
        
        # Output as JSON
        result = {
            'duration_seconds': float(duration),
            'bpm': float(tempo),
            'key': key,
            'tempo': get_tempo_description(tempo),
            'beat_times': [float(t) for t in beat_times.tolist()],
            'beat_count': len(beat_times),
            'vocal_segments': vocal_segments,
            'vocal_segment_count': len(vocal_segments),
            'spectral_centroid': spectral_centroid,
            'zero_crossing_rate': zero_crossing_rate,
            'sample_rate': sr,
            'success': True
        }
        
        return result
        
    except Exception as e:
        return {
            'success': False,
            'error': str(e),
            'error_type': type(e).__name__
        }


def main():
    """Command-line interface"""
    if len(sys.argv) != 2:
        print(json.dumps({
            'success': False,
            'error': 'Usage: python analyzer.py <audio_file_path>'
        }))
        sys.exit(1)
    
    file_path = sys.argv[1]
    result = analyze_audio(file_path)
    
    # Output JSON
    print(json.dumps(result, indent=2))
    
    # Exit with appropriate code
    sys.exit(0 if result.get('success', False) else 1)


if __name__ == '__main__':
    main()

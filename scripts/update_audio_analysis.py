#!/usr/bin/env python3
"""
Update audio analysis for all songs in the database
Runs analyzer.py on each song's instrumental track and updates the database
"""
import sqlite3
import json
import subprocess
import sys
import os

# Database path
DB_PATH = os.path.join(os.path.dirname(__file__), '..', 'data', 'trackstudio.db')
ANALYZER_PATH = os.path.join(os.path.dirname(__file__), '..', 'pkg', 'audio', 'analyzer.py')

def analyze_audio_file(file_path):
    """Run analyzer.py on a file and return results"""
    try:
        result = subprocess.run(
            ['python3', ANALYZER_PATH, file_path],
            capture_output=True,
            text=True,
            check=True
        )
        return json.loads(result.stdout)
    except subprocess.CalledProcessError as e:
        print(f"Error analyzing {file_path}: {e.stderr}")
        return None
    except json.JSONDecodeError as e:
        print(f"Error parsing JSON for {file_path}: {e}")
        return None

def update_song_analysis(conn, song_id, analysis):
    """Update song record with analysis results"""
    if not analysis or not analysis.get('success'):
        print(f"  ❌ Skipping song {song_id} - analysis failed")
        return False
    
    cursor = conn.cursor()
    cursor.execute("""
        UPDATE songs 
        SET 
            duration_seconds = ?,
            bpm = ?,
            key = ?,
            tempo = ?,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    """, (
        analysis['duration_seconds'],
        analysis['bpm'],
        analysis['key'],
        analysis['tempo'],
        song_id
    ))
    
    conn.commit()
    return True

def main():
    print("🎵 Audio Analysis Update Script\n")
    print("=" * 60)
    
    # Connect to database
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    
    # Get all songs with their audio paths
    cursor.execute("""
        SELECT id, title, music_stem_path, bpm, key
        FROM songs 
        ORDER BY id
    """)
    
    songs = cursor.fetchall()
    
    if not songs:
        print("No songs found in database")
        return
    
    print(f"\nFound {len(songs)} songs to analyze\n")
    
    success_count = 0
    fail_count = 0
    
    for song_id, title, music_stem_path, current_bpm, current_key in songs:
        print(f"📀 Song {song_id}: {title}")
        bpm_str = f"{current_bpm:.2f}" if current_bpm else "0"
        print(f"   Current: BPM={bpm_str}, Key={current_key or 'N/A'}")
        
        if not music_stem_path or not os.path.exists(music_stem_path):
            print(f"   ❌ Audio file not found: {music_stem_path}")
            fail_count += 1
            continue
        
        # Analyze audio
        print(f"   🔍 Analyzing: {os.path.basename(music_stem_path)}")
        analysis = analyze_audio_file(music_stem_path)
        
        if analysis and analysis.get('success'):
            # Update database
            if update_song_analysis(conn, song_id, analysis):
                print(f"   ✅ Updated: BPM={analysis['bpm']:.2f}, Key={analysis['key']}, Tempo={analysis['tempo']}, Duration={analysis['duration_seconds']:.2f}s")
                success_count += 1
            else:
                print(f"   ❌ Failed to update database")
                fail_count += 1
        else:
            error_msg = analysis.get('error', 'Unknown error') if analysis else 'Analysis failed'
            print(f"   ❌ Analysis failed: {error_msg}")
            fail_count += 1
        
        print()
    
    conn.close()
    
    print("=" * 60)
    print(f"\n✨ Complete!")
    print(f"   ✅ Success: {success_count} songs")
    print(f"   ❌ Failed: {fail_count} songs")
    print()

if __name__ == '__main__':
    main()

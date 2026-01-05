#!/usr/bin/env python3
"""
Scan storage/videos directory and populate videos table with existing video files.
Extracts metadata from filenames and video properties.
"""

import os
import sqlite3
import json
from pathlib import Path
import subprocess
import sys

def get_video_duration(video_path):
    """Get video duration using ffprobe"""
    try:
        result = subprocess.run(
            ['ffprobe', '-v', 'error', '-show_entries', 'format=duration', 
             '-of', 'json', video_path],
            capture_output=True,
            text=True,
            check=True
        )
        data = json.loads(result.stdout)
        return float(data['format']['duration'])
    except Exception as e:
        print(f"Warning: Could not get duration for {video_path}: {e}")
        return None

def get_video_resolution(video_path):
    """Get video resolution using ffprobe"""
    try:
        result = subprocess.run(
            ['ffprobe', '-v', 'error', '-select_streams', 'v:0',
             '-show_entries', 'stream=width,height', '-of', 'json', video_path],
            capture_output=True,
            text=True,
            check=True
        )
        data = json.loads(result.stdout)
        width = data['streams'][0]['width']
        height = data['streams'][0]['height']
        
        # Determine resolution label
        if height >= 2160:
            return '4k'
        elif height >= 1080:
            return '1080p'
        elif height >= 720:
            return '720p'
        else:
            return '480p'
    except Exception as e:
        print(f"Warning: Could not get resolution for {video_path}: {e}")
        return '4k'  # Default assumption

def normalize_title(title):
    """Normalize title for matching - remove special chars, lowercase"""
    import re
    # Remove anything in parentheses
    title = re.sub(r'\([^)]*\)', '', title)
    # Remove special characters and extra spaces
    title = re.sub(r'[^a-z0-9\s]', '', title.lower())
    # Collapse multiple spaces
    title = ' '.join(title.split())
    return title.strip()

def find_song_by_filename(cursor, filename):
    """Find song ID by matching filename to song title"""
    # Remove .mp4 extension
    base_name = filename.replace('.mp4', '')
    # Replace underscores with spaces
    search_title = base_name.replace('_', ' ')
    
    normalized_search = normalize_title(search_title)
    
    # Get all songs
    cursor.execute("SELECT id, title FROM songs")
    songs = cursor.fetchall()
    
    for song_id, song_title in songs:
        normalized_song = normalize_title(song_title)
        
        # Check for exact match
        if normalized_search == normalized_song:
            return song_id
        
        # Check if one contains the other
        if normalized_search in normalized_song or normalized_song in normalized_search:
            return song_id
    
    return None

def populate_videos_table(db_path, storage_dir):
    """Scan videos directory and populate videos table"""
    
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    
    # Get all video files
    videos_dir = os.path.join(storage_dir, 'videos')
    if not os.path.exists(videos_dir):
        print(f"Videos directory not found: {videos_dir}")
        return
    
    video_files = list(Path(videos_dir).rglob('*.mp4'))
    print(f"Found {len(video_files)} video files")
    
    inserted = 0
    skipped = 0
    errors = 0
    songs_created = 0
    
    for video_path in video_files:
        video_path_str = str(video_path)
        filename = video_path.name
        
        # Find song by matching filename to title
        song_id = find_song_by_filename(cursor, filename)
        
        # If no song found, create a placeholder song record
        if song_id is None:
            song_title = filename.replace('.mp4', '').replace('_', ' ')
            print(f"Creating placeholder song for: {song_title}")
            
            try:
                cursor.execute("""
                    INSERT INTO songs 
                    (title, artist_name, vocals_stem_path, music_stem_path, lyrics)
                    VALUES (?, 'Tristan Hart', '', '', 'Lyrics not available')
                """, (song_title,))
                song_id = cursor.lastrowid
                songs_created += 1
                print(f"  → Created song ID {song_id}")
            except Exception as e:
                print(f"  ✗ Failed to create song: {e}")
                errors += 1
                continue
        
        # Check if song exists
        cursor.execute("SELECT id FROM songs WHERE id = ?", (song_id,))
        if not cursor.fetchone():
            print(f"Skipping {video_path.name} - song ID {song_id} not found in database")
            skipped += 1
            continue
        
        # Get video metadata
        file_size = os.path.getsize(video_path_str)
        duration = get_video_duration(video_path_str)
        resolution = get_video_resolution(video_path_str)
        
        # Get file modification time as rendered_at
        mtime = os.path.getmtime(video_path_str)
        from datetime import datetime
        rendered_at = datetime.fromtimestamp(mtime).isoformat()
        
        # Get metadata from song
        cursor.execute("""
            SELECT genre, bpm, key, tempo FROM songs WHERE id = ?
        """, (song_id,))
        song_metadata = cursor.fetchone()
        genre = song_metadata[0] if song_metadata else None
        bpm = song_metadata[1] if song_metadata else None
        key = song_metadata[2] if song_metadata else None
        tempo = song_metadata[3] if song_metadata else None
        
        # Make path relative to storage directory
        rel_path = os.path.relpath(video_path_str, storage_dir)
        
        try:
            # Insert into videos table
            cursor.execute("""
                INSERT INTO videos 
                (song_id, video_file_path, resolution, duration_seconds, 
                 file_size_bytes, status, rendered_at, genre, bpm, key, tempo)
                VALUES (?, ?, ?, ?, ?, 'completed', ?, ?, ?, ?, ?)
            """, (song_id, rel_path, resolution, duration, file_size, rendered_at,
                  genre, bpm, key, tempo))
            
            inserted += 1
            print(f"✓ Inserted: {video_path.name} (Song {song_id}, {resolution}, {file_size/1024/1024:.1f}MB)")
            
        except sqlite3.IntegrityError:
            # Video already exists
            skipped += 1
            print(f"⊗ Skipped: {video_path.name} (already in database)")
        except Exception as e:
            errors += 1
            print(f"✗ Error: {video_path.name} - {e}")
    
    conn.commit()
    conn.close()
    
    print(f"\n{'='*60}")
    print(f"Summary:")
    print(f"  Songs created: {songs_created}")
    print(f"  Videos inserted: {inserted}")
    print(f"  Skipped:  {skipped}")
    print(f"  Errors:   {errors}")
    print(f"{'='*60}")

if __name__ == '__main__':
    # Determine paths
    script_dir = os.path.dirname(os.path.abspath(__file__))
    orchestrator_dir = os.path.dirname(script_dir)
    
    # Use the correct database location
    db_path = os.path.join(orchestrator_dir, 'data', 'trackstudio.db')
    
    if not os.path.exists(db_path):
        print(f"Error: Database not found at {db_path}")
        sys.exit(1)
    
    # Storage directory
    storage_dir = os.path.join(orchestrator_dir, 'bin', 'storage')
    if not os.path.exists(storage_dir):
        storage_dir = os.path.join(orchestrator_dir, 'storage')
    
    print(f"Database: {db_path}")
    print(f"Storage:  {storage_dir}\n")
    
    populate_videos_table(db_path, storage_dir)

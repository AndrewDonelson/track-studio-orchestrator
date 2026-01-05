#!/usr/bin/env python3
"""
Fix image filenames to match new naming convention:
- Verses remain numbered: bg-verse-1.png, bg-verse-2.png, etc.
- Reusable sections not numbered: bg-chorus.png, bg-prechorus.png, bg-bridge.png, bg-intro.png, bg-outro.png
"""

import os
import sqlite3
from pathlib import Path

# Configuration
IMAGES_DIR = Path("bin/storage/images")
DB_PATH = "data/trackstudio.db"

# Mapping of old names to new names (without song-specific directory)
RENAME_MAP = {
    "bg-chorus-1.png": "bg-chorus.png",
    "bg-pre-chorus-1.png": "bg-prechorus.png",  # Note: also removes hyphen from "pre-chorus"
    "bg-final-chorus-1.png": "bg-chorus.png",
    "bg-bridge-1.png": "bg-bridge.png",
    "bg-outro-1.png": "bg-outro.png",
}

def main():
    renamed_count = 0
    db_updates = []
    
    # Scan all song directories
    if not IMAGES_DIR.exists():
        print(f"❌ Images directory not found: {IMAGES_DIR}")
        return
    
    for song_dir in IMAGES_DIR.iterdir():
        if not song_dir.is_dir() or not song_dir.name.startswith("song_"):
            continue
        
        print(f"\n📁 Processing {song_dir.name}...")
        
        for old_name, new_name in RENAME_MAP.items():
            old_path = song_dir / old_name
            new_path = song_dir / new_name
            
            if old_path.exists():
                # If target already exists, we need to decide what to do
                if new_path.exists():
                    print(f"  ⚠️  {new_name} already exists, skipping {old_name}")
                    continue
                
                # Rename the file
                old_path.rename(new_path)
                print(f"  ✅ Renamed: {old_name} → {new_name}")
                renamed_count += 1
                
                # Track database update needed
                old_db_path = f"{song_dir.name}/{old_name}"
                new_db_path = f"{song_dir.name}/{new_name}"
                db_updates.append((new_db_path, old_db_path))
    
    print(f"\n📊 Renamed {renamed_count} files")
    
    # Update database
    if db_updates:
        print(f"\n🗄️  Updating database records...")
        try:
            conn = sqlite3.connect(DB_PATH)
            cursor = conn.cursor()
            
            for new_path, old_path in db_updates:
                cursor.execute("""
                    UPDATE generated_images 
                    SET image_path = ? 
                    WHERE image_path = ?
                """, (new_path, old_path))
            
            conn.commit()
            print(f"  ✅ Updated {cursor.rowcount} database records")
            conn.close()
        except Exception as e:
            print(f"  ❌ Database error: {e}")
    
    print("\n✅ Image filename fix complete!")

if __name__ == "__main__":
    main()

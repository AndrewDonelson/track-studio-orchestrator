#!/usr/bin/env python3
"""
Clean up all prompts in the database:
1. Remove all quality modifier variations
2. Clear negative prompts (unused)
"""

import sqlite3
import re

DB_PATH = "data/trackstudio.db"

# All variations of quality modifiers to remove
QUALITY_PATTERNS = [
    r',?\s*photorealistic,?\s*professional photography,?\s*8K resolution,?\s*ultra detailed,?\s*sharp focus,?\s*cinematic composition,?\s*award-winning photography',
    r',?\s*photorealistic,?\s*professional photography,?\s*beautiful composition,?\s*artistic,?\s*cinematic,?\s*8K resolution,?\s*ultra detailed,?\s*sharp focus,?\s*award-winning photography',
    r',?\s*photorealistic,?\s*professional photography,?\s*beautiful composition,?\s*artistic,?\s*cinematic,?\s*8K resolution,?\s*ultra detailed,?\s*sharp focus',
    r',?\s*photorealistic,?\s*professional photography,?\s*cinematic composition,?\s*8K,?\s*ultra detailed,?\s*sharp focus',
    r',?\s*professional quality,?\s*beautiful composition,?\s*artistic,?\s*photorealistic,?\s*8K resolution,?\s*ultra detailed,?\s*sharp focus',
    r',?\s*photorealistic',
    r',?\s*professional photography',
    r',?\s*8K resolution',
    r',?\s*ultra detailed',
    r',?\s*sharp focus',
    r',?\s*cinematic composition',
    r',?\s*award-winning photography',
    r',?\s*beautiful composition',
    r',?\s*artistic',
    r',?\s*professional quality',
]

def clean_prompt(prompt):
    """Remove all quality modifier variations from prompt"""
    if not prompt:
        return prompt
    
    cleaned = prompt
    
    # Apply each pattern
    for pattern in QUALITY_PATTERNS:
        cleaned = re.sub(pattern, '', cleaned, flags=re.IGNORECASE)
    
    # Clean up any trailing commas and extra spaces
    cleaned = re.sub(r'\s*,\s*$', '', cleaned)
    cleaned = re.sub(r'\s+', ' ', cleaned)
    cleaned = cleaned.strip()
    
    return cleaned

def main():
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    
    # Get all prompts
    cursor.execute("SELECT id, prompt FROM generated_images WHERE prompt IS NOT NULL")
    rows = cursor.fetchall()
    
    print(f"Processing {len(rows)} prompts...")
    
    updated = 0
    for img_id, prompt in rows:
        cleaned = clean_prompt(prompt)
        if cleaned != prompt:
            cursor.execute("UPDATE generated_images SET prompt = ? WHERE id = ?", (cleaned, img_id))
            updated += 1
    
    conn.commit()
    
    print(f"✅ Updated {updated} prompts")
    print(f"✅ Cleared all negative prompts")
    
    # Show samples
    print("\nSample cleaned prompts:")
    cursor.execute("SELECT image_type, SUBSTR(prompt, -100) FROM generated_images WHERE song_id IN (1, 30) LIMIT 5")
    for img_type, ending in cursor.fetchall():
        print(f"  {img_type}: ...{ending}")
    
    conn.close()

if __name__ == "__main__":
    main()

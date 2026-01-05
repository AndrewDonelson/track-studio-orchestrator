-- Migration: Add videos table to persist rendered videos
-- This ensures video records survive queue cleanup operations

CREATE TABLE IF NOT EXISTS videos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    song_id INTEGER NOT NULL,
    
    -- File paths
    video_file_path TEXT NOT NULL UNIQUE,
    thumbnail_path TEXT,
    
    -- Video metadata
    resolution TEXT,  -- '4k', '1080p', '720p', '480p'
    duration_seconds REAL,
    file_size_bytes INTEGER,
    fps INTEGER DEFAULT 60,
    
    -- Rendering details
    background_style TEXT,
    spectrum_color TEXT,
    has_karaoke BOOLEAN DEFAULT 1,
    
    -- Status tracking
    status TEXT DEFAULT 'completed',  -- 'completed', 'archived', 'deleted'
    
    -- Timestamps
    rendered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
);

-- Index for fast lookups
CREATE INDEX IF NOT EXISTS idx_videos_song ON videos(song_id);
CREATE INDEX IF NOT EXISTS idx_videos_status ON videos(status);
CREATE INDEX IF NOT EXISTS idx_videos_path ON videos(video_file_path);

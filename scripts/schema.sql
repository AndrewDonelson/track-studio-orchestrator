-- Track Studio Database Schema
-- SQLite database for song video generation

-- Artists table
CREATE TABLE IF NOT EXISTS artists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    bio TEXT,
    website TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Albums table
CREATE TABLE IF NOT EXISTS albums (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    artist_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    release_year INTEGER,
    cover_art_path TEXT,
    youtube_playlist_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (artist_id) REFERENCES artists(id)
);

-- Songs table
CREATE TABLE IF NOT EXISTS songs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    album_id INTEGER,
    title TEXT NOT NULL,
    artist_name TEXT NOT NULL DEFAULT 'Tristan Hart',
    genre TEXT,
    
    -- Audio stems
    vocals_stem_path TEXT NOT NULL,
    music_stem_path TEXT NOT NULL,
    mixed_audio_path TEXT,
    
    -- Metadata
    metadata_file_path TEXT,
    
    -- Lyrics
    lyrics TEXT NOT NULL,
    lyrics_display TEXT,  -- JSON
    lyrics_sections TEXT, -- JSON
    
    -- Audio analysis
    bpm REAL,
    key TEXT,
    tempo TEXT,
    duration_seconds REAL,
    
    -- Vocal timing
    vocal_timing TEXT, -- JSON
    
    -- Branding
    brand_logo_path TEXT,
    copyright_text TEXT DEFAULT 'All content Copyright 2017-2026 Nlaak Studios',
    
    -- Video settings
    background_style TEXT DEFAULT 'cinematic',
    spectrum_color TEXT DEFAULT 'rainbow',
    spectrum_opacity REAL DEFAULT 0.25,
    target_resolution TEXT DEFAULT '4k',
    show_metadata BOOLEAN DEFAULT 1,  -- Show BPM, Key, Tempo at top
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (album_id) REFERENCES albums(id)
);

-- Queue table
CREATE TABLE IF NOT EXISTS queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    song_id INTEGER NOT NULL,
    
    status TEXT NOT NULL DEFAULT 'queued',
    priority INTEGER DEFAULT 0,
    
    current_step TEXT,
    progress INTEGER DEFAULT 0,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    
    video_file_path TEXT,
    video_file_size INTEGER,
    thumbnail_path TEXT,
    
    flag TEXT, -- User-reported issues: 'image_issue', 'lyrics_issue', 'timing_issue', or NULL
    
    queued_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    
    FOREIGN KEY (song_id) REFERENCES songs(id)
);

-- YouTube uploads table
CREATE TABLE IF NOT EXISTS youtube_uploads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    queue_id INTEGER NOT NULL,
    song_id INTEGER NOT NULL,
    
    youtube_video_id TEXT UNIQUE,
    youtube_url TEXT,
    title TEXT NOT NULL,
    description TEXT,
    tags TEXT,
    category_id INTEGER DEFAULT 10,
    
    privacy_status TEXT DEFAULT 'public',
    
    upload_started_at TIMESTAMP,
    upload_completed_at TIMESTAMP,
    
    views INTEGER DEFAULT 0,
    likes INTEGER DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (queue_id) REFERENCES queue(id),
    FOREIGN KEY (song_id) REFERENCES songs(id)
);

-- Processing logs table
CREATE TABLE IF NOT EXISTS processing_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    queue_id INTEGER NOT NULL,
    step TEXT NOT NULL,
    status TEXT NOT NULL,
    message TEXT,
    duration_seconds REAL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (queue_id) REFERENCES queue(id)
);

-- Generated images table for tracking AI-generated images
CREATE TABLE IF NOT EXISTS generated_images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    song_id INTEGER NOT NULL,
    queue_id INTEGER,
    
    image_path TEXT NOT NULL,
    prompt TEXT NOT NULL,
    negative_prompt TEXT,
    image_type TEXT NOT NULL, -- 'background', 'scene', 'thumbnail', etc.
    sequence_number INTEGER, -- For ordering scene images
    
    width INTEGER,
    height INTEGER,
    model TEXT, -- AI model used
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (song_id) REFERENCES songs(id),
    FOREIGN KEY (queue_id) REFERENCES queue(id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_songs_album ON songs(album_id);
CREATE INDEX IF NOT EXISTS idx_queue_status ON queue(status);
CREATE INDEX IF NOT EXISTS idx_queue_song ON queue(song_id);
CREATE INDEX IF NOT EXISTS idx_youtube_video_id ON youtube_uploads(youtube_video_id);
CREATE INDEX IF NOT EXISTS idx_logs_queue ON processing_logs(queue_id);
CREATE INDEX IF NOT EXISTS idx_images_song ON generated_images(song_id);
CREATE INDEX IF NOT EXISTS idx_images_queue ON generated_images(queue_id);

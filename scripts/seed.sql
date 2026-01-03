-- Seed data for Track Studio

-- Insert default artist
INSERT INTO artists (name, bio, website) VALUES 
('Tristan Hart', 'Electronic and ambient music artist', 'https://tristanhart.com');

-- Insert default album
INSERT INTO albums (artist_id, title, release_year) VALUES 
(1, 'Demo Collection', 2026);

-- Insert sample song
INSERT INTO songs (
    album_id, 
    title, 
    artist_name, 
    genre,
    vocals_stem_path,
    music_stem_path,
    lyrics,
    bpm,
    duration_seconds,
    background_style,
    spectrum_color,
    spectrum_opacity,
    target_resolution
) VALUES (
    1,
    'Test Song',
    'Tristan Hart',
    'Electronic',
    'storage/songs/test/vocals.wav',
    'storage/songs/test/music.wav',
    'This is a test song
With multiple lines
For testing purposes',
    120.0,
    180.0,
    'cinematic',
    'rainbow',
    0.25,
    '4k'
);

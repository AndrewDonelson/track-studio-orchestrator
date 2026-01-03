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
    mixed_audio_path,
    lyrics,
    bpm,
    duration_seconds,
    background_style,
    spectrum_color,
    spectrum_opacity,
    target_resolution
) VALUES (
    1,
    'Land of Love',
    'Tristan Hart',
    'Electronic',
    '/home/andrew/Development/Fullstack-Projects/TrackStudio/test-files/Land of Love (Cover Hung) (Vocals).wav',
    '/home/andrew/Development/Fullstack-Projects/TrackStudio/test-files/Land of Love (Cover Hung) (Instrumental).wav',
    '/home/andrew/Development/Fullstack-Projects/TrackStudio/test-files/Land of Love (Cover Hung) (Vocals).wav',
    'In the land of love
Where hearts collide
We dance together
Side by side

Through the night we fly
Underneath the stars
In this land of love
Forever ours',
    120.0,
    180.0,
    'cinematic',
    'rainbow',
    0.25,
    '4k'
);

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
    target_resolution,
    show_metadata
) VALUES (
    1,
    'Land of Love (Cover Hung)',
    'Tristan Hart',
    'Romantic Pop',
    '/home/andrew/Development/Fullstack-Projects/TrackStudio/test-files/Land of Love (Cover Hung) (Vocals).wav',
    '/home/andrew/Development/Fullstack-Projects/TrackStudio/test-files/Land of Love (Cover Hung) (Instrumental).wav',
    '/home/andrew/Development/Fullstack-Projects/TrackStudio/test-files/Land of Love (Cover Hung) (Vocals).wav',
    '[Verse 1]
In the city streets of Saigon, where the night air whispers low
I saw you standing there, like a vision from above
Your eyes, like the Mekong River, flowing deep and wide
And I knew in that moment, my heart would be yours to reside

[Pre-Chorus]
Oh, my love, with skin as smooth as silk
You''re the moonlight on the Perfume River, my heart''s only milk
In your eyes, I see a love so true
A love that''s worth waiting for, a love that''s meant for me and you

[Chorus]
I''ve been searching for a love like yours, in every place I roam
But none have ever touched my heart, like the way you make me feel at home
In your arms, is where I belong
You are my land of love, my heart beats for you alone

[Verse 2]
We''d walk along the beach, in Nha Trang''s golden light
And I''d tell you stories of my dreams, and the love we''d ignite
You''d smile and laugh, and my heart would skip a beat
And I knew in that moment, our love would forever be unique

[Pre-Chorus]
Oh, my love, with skin as smooth as silk
You''re the moonlight on the Perfume River, my heart''s only milk
In your eyes, I see a love so true
A love that''s worth waiting for, a love that''s meant for me and you

[Bridge]
We''ll dance beneath the stars, on a warm summer night
With gentle music playing, and incense drifting through the air tonight
And I''ll whisper "I love you" softly in your ear
And you''ll whisper back, "I love you" for me to hear

[Chorus]
I''ve been searching for a love like yours, in every place I roam
But none have ever touched my heart, like the way you make me feel at home
In your arms, is where I belong
You are my land of love, my heart beats for you alone',
    0.0,
    0.0,
    'cinematic',
    'rainbow',
    0.25,
    '4k',
    1
);

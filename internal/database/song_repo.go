package database

import (
	"database/sql"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
)

// SongRepository handles song database operations
type SongRepository struct {
	db *sql.DB
}

// NewSongRepository creates a new song repository
func NewSongRepository(db *sql.DB) *SongRepository {
	return &SongRepository{db: db}
}

// GetAll returns all songs
func (r *SongRepository) GetAll() ([]models.Song, error) {
	query := `SELECT id, album_id, title, artist_name, genre, 
		vocals_stem_path, music_stem_path, 
		COALESCE(mixed_audio_path, '') as mixed_audio_path, 
		COALESCE(metadata_file_path, '') as metadata_file_path,
		lyrics, 
		COALESCE(lyrics_karaoke, '') as lyrics_karaoke,
		COALESCE(lyrics_display, '') as lyrics_display, 
		COALESCE(lyrics_sections, '') as lyrics_sections,
		COALESCE(bpm, 0) as bpm, 
		COALESCE(key, '') as key, 
		COALESCE(tempo, '') as tempo, 
		COALESCE(duration_seconds, 0) as duration_seconds, 
		COALESCE(vocal_timing, '') as vocal_timing,
		COALESCE(brand_logo_path, '') as brand_logo_path, 
		COALESCE(copyright_text, '') as copyright_text,
		COALESCE(background_style, 'cinematic') as background_style, 
		COALESCE(spectrum_color, 'rainbow') as spectrum_color, 
		COALESCE(spectrum_opacity, 0.25) as spectrum_opacity, 
		COALESCE(target_resolution, '4k') as target_resolution,
		created_at, updated_at
		FROM songs ORDER BY created_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []models.Song
	for rows.Next() {
		var s models.Song
		err := rows.Scan(
			&s.ID, &s.AlbumID, &s.Title, &s.ArtistName, &s.Genre,
			&s.VocalsStemPath, &s.MusicStemPath, &s.MixedAudioPath, &s.MetadataPath,
			&s.Lyrics, &s.LyricsKaraoke, &s.LyricsDisplay, &s.LyricsSections,
			&s.BPM, &s.Key, &s.Tempo, &s.DurationSeconds, &s.VocalTiming,
			&s.BrandLogoPath, &s.CopyrightText,
			&s.BackgroundStyle, &s.SpectrumColor, &s.SpectrumOpacity, &s.TargetResolution,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		songs = append(songs, s)
	}

	return songs, nil
}

// GetByID returns a song by ID
func (r *SongRepository) GetByID(id int) (*models.Song, error) {
	query := `SELECT id, album_id, title, artist_name, genre,
		vocals_stem_path, music_stem_path, 
		COALESCE(mixed_audio_path, '') as mixed_audio_path, 
		COALESCE(metadata_file_path, '') as metadata_file_path,
		lyrics, 
		COALESCE(lyrics_karaoke, '') as lyrics_karaoke,
		COALESCE(lyrics_display, '') as lyrics_display, 
		COALESCE(lyrics_sections, '') as lyrics_sections,
		COALESCE(bpm, 0) as bpm, 
		COALESCE(key, '') as key, 
		COALESCE(tempo, '') as tempo, 
		COALESCE(duration_seconds, 0) as duration_seconds, 
		COALESCE(vocal_timing, '') as vocal_timing,
		COALESCE(brand_logo_path, '') as brand_logo_path, 
		COALESCE(copyright_text, '') as copyright_text,
		COALESCE(background_style, 'cinematic') as background_style, 
		COALESCE(spectrum_color, 'rainbow') as spectrum_color, 
		COALESCE(spectrum_opacity, 0.25) as spectrum_opacity, 
		COALESCE(target_resolution, '4k') as target_resolution,
		created_at, updated_at
		FROM songs WHERE id = ?`

	var s models.Song
	err := r.db.QueryRow(query, id).Scan(
		&s.ID, &s.AlbumID, &s.Title, &s.ArtistName, &s.Genre,
		&s.VocalsStemPath, &s.MusicStemPath, &s.MixedAudioPath, &s.MetadataPath,
		&s.Lyrics, &s.LyricsKaraoke, &s.LyricsDisplay, &s.LyricsSections,
		&s.BPM, &s.Key, &s.Tempo, &s.DurationSeconds, &s.VocalTiming,
		&s.BrandLogoPath, &s.CopyrightText,
		&s.BackgroundStyle, &s.SpectrumColor, &s.SpectrumOpacity, &s.TargetResolution,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// Create creates a new song
func (r *SongRepository) Create(song *models.Song) error {
	query := `INSERT INTO songs (album_id, title, artist_name, genre,
		vocals_stem_path, music_stem_path, mixed_audio_path, metadata_file_path,
		lyrics, lyrics_karaoke, lyrics_display, lyrics_sections,
		bpm, key, tempo, duration_seconds, vocal_timing,
		brand_logo_path, copyright_text,
		background_style, spectrum_color, spectrum_opacity, target_resolution)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.Exec(query,
		song.AlbumID, song.Title, song.ArtistName, song.Genre,
		song.VocalsStemPath, song.MusicStemPath, song.MixedAudioPath, song.MetadataPath,
		song.Lyrics, song.LyricsKaraoke, song.LyricsDisplay, song.LyricsSections,
		song.BPM, song.Key, song.Tempo, song.DurationSeconds, song.VocalTiming,
		song.BrandLogoPath, song.CopyrightText,
		song.BackgroundStyle, song.SpectrumColor, song.SpectrumOpacity, song.TargetResolution,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	song.ID = int(id)
	return nil
}

// Update updates an existing song
func (r *SongRepository) Update(song *models.Song) error {
	query := `UPDATE songs SET album_id=?, title=?, artist_name=?, genre=?,
		vocals_stem_path=?, music_stem_path=?, mixed_audio_path=?, metadata_file_path=?,
		lyrics=?, lyrics_karaoke=?, lyrics_display=?, lyrics_sections=?,
		bpm=?, key=?, tempo=?, duration_seconds=?, vocal_timing=?,
		brand_logo_path=?, copyright_text=?,
		background_style=?, spectrum_color=?, spectrum_opacity=?, target_resolution=?,
		updated_at=CURRENT_TIMESTAMP
		WHERE id=?`

	_, err := r.db.Exec(query,
		song.AlbumID, song.Title, song.ArtistName, song.Genre,
		song.VocalsStemPath, song.MusicStemPath, song.MixedAudioPath, song.MetadataPath,
		song.Lyrics, song.LyricsKaraoke, song.LyricsDisplay, song.LyricsSections,
		song.BPM, song.Key, song.Tempo, song.DurationSeconds, song.VocalTiming,
		song.BrandLogoPath, song.CopyrightText,
		song.BackgroundStyle, song.SpectrumColor, song.SpectrumOpacity, song.TargetResolution,
		song.ID,
	)
	return err
}

// Delete deletes a song
func (r *SongRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM songs WHERE id=?", id)
	return err
}

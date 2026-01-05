package database

import (
	"database/sql"
	"time"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
)

type VideoRepository struct {
	db *sql.DB
}

func NewVideoRepository(db *sql.DB) *VideoRepository {
	return &VideoRepository{db: db}
}

// GetAll returns all videos
func (r *VideoRepository) GetAll() ([]models.Video, error) {
	query := `
		SELECT v.id, v.song_id, v.video_file_path, v.thumbnail_path, 
		       v.resolution, v.duration_seconds, v.file_size_bytes, v.fps,
		       v.background_style, v.spectrum_color, v.has_karaoke,
		       v.status, v.rendered_at, v.created_at,
		       v.genre, v.bpm, v.key, v.tempo, v.flag,
		       s.title, s.artist_name
		FROM videos v
		JOIN songs s ON v.song_id = s.id
		WHERE v.status = 'completed'
		ORDER BY v.rendered_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	videos := []models.Video{}
	for rows.Next() {
		var v models.Video
		var renderedAt, createdAt string

		err := rows.Scan(
			&v.ID, &v.SongID, &v.VideoFilePath, &v.ThumbnailPath,
			&v.Resolution, &v.DurationSeconds, &v.FileSizeBytes, &v.FPS,
			&v.BackgroundStyle, &v.SpectrumColor, &v.HasKaraoke,
			&v.Status, &renderedAt, &createdAt,
			&v.Genre, &v.BPM, &v.Key, &v.Tempo, &v.Flag,
			&v.SongTitle, &v.ArtistName,
		)
		if err != nil {
			return nil, err
		}

		v.RenderedAt, _ = time.Parse(time.RFC3339, renderedAt)
		v.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

		videos = append(videos, v)
	}

	return videos, nil
}

// GetBySongID returns all videos for a song
func (r *VideoRepository) GetBySongID(songID int) ([]models.Video, error) {
	query := `
		SELECT v.id, v.song_id, v.video_file_path, v.thumbnail_path, 
		       v.resolution, v.duration_seconds, v.file_size_bytes, v.fps,
		       v.background_style, v.spectrum_color, v.has_karaoke,
		       v.status, v.rendered_at, v.created_at,
		       v.genre, v.bpm, v.key, v.tempo, v.flag,
		       s.title, s.artist_name
		FROM videos v
		JOIN songs s ON v.song_id = s.id
		WHERE v.song_id = ? AND v.status = 'completed'
		ORDER BY v.rendered_at DESC
	`

	rows, err := r.db.Query(query, songID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	videos := []models.Video{}
	for rows.Next() {
		var v models.Video
		var renderedAt, createdAt string

		err := rows.Scan(
			&v.ID, &v.SongID, &v.VideoFilePath, &v.ThumbnailPath,
			&v.Resolution, &v.DurationSeconds, &v.FileSizeBytes, &v.FPS,
			&v.BackgroundStyle, &v.SpectrumColor, &v.HasKaraoke,
			&v.Status, &renderedAt, &createdAt,
			&v.Genre, &v.BPM, &v.Key, &v.Tempo, &v.Flag,
			&v.SongTitle, &v.ArtistName,
		)
		if err != nil {
			return nil, err
		}

		v.RenderedAt, _ = time.Parse(time.RFC3339, renderedAt)
		v.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

		videos = append(videos, v)
	}

	return videos, nil
}

// Create inserts a new video record
func (r *VideoRepository) Create(video *models.Video) error {
	query := `
		INSERT INTO videos 
		(song_id, video_file_path, thumbnail_path, resolution, duration_seconds, 
		 file_size_bytes, fps, background_style, spectrum_color, has_karaoke, status, rendered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		video.SongID,
		video.VideoFilePath,
		video.ThumbnailPath,
		video.Resolution,
		video.DurationSeconds,
		video.FileSizeBytes,
		video.FPS,
		video.BackgroundStyle,
		video.SpectrumColor,
		video.HasKaraoke,
		video.Status,
		video.RenderedAt,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	video.ID = int(id)
	return nil
}

// CreateOrUpdate inserts a new video or updates existing one for the same song
func (r *VideoRepository) CreateOrUpdate(video *models.Video) error {
	// Check if ANY video already exists for this song (regardless of status)
	var existingID int
	query := `SELECT id FROM videos WHERE song_id = ? ORDER BY created_at DESC LIMIT 1`
	err := r.db.QueryRow(query, video.SongID).Scan(&existingID)

	if err == nil {
		// Video exists, update it
		updateQuery := `
			UPDATE videos 
			SET video_file_path = ?, thumbnail_path = ?, resolution = ?, 
			    duration_seconds = ?, file_size_bytes = ?, fps = ?,
			    background_style = ?, spectrum_color = ?, has_karaoke = ?,
			    status = ?, rendered_at = ?,
			    genre = ?, bpm = ?, key = ?, tempo = ?
			WHERE id = ?
		`

		_, err := r.db.Exec(
			updateQuery,
			video.VideoFilePath,
			video.ThumbnailPath,
			video.Resolution,
			video.DurationSeconds,
			video.FileSizeBytes,
			video.FPS,
			video.BackgroundStyle,
			video.SpectrumColor,
			video.HasKaraoke,
			video.Status,
			video.RenderedAt,
			video.Genre,
			video.BPM,
			video.Key,
			video.Tempo,
			existingID,
		)
		if err != nil {
			return err
		}
		video.ID = existingID
		return nil
	}

	if err != sql.ErrNoRows {
		return err
	}

	// No existing video, create new record
	return r.Create(video)
}

// Delete marks a video as deleted (soft delete)
func (r *VideoRepository) Delete(id int) error {
	_, err := r.db.Exec("UPDATE videos SET status = 'deleted' WHERE id = ?", id)
	return err
}

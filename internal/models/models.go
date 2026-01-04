package models

import "time"

// Artist represents a music artist
type Artist struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Bio       string    `json:"bio" db:"bio"`
	Website   string    `json:"website" db:"website"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Album represents a music album
type Album struct {
	ID                int       `json:"id" db:"id"`
	ArtistID          int       `json:"artist_id" db:"artist_id"`
	Title             string    `json:"title" db:"title"`
	ReleaseYear       int       `json:"release_year" db:"release_year"`
	CoverArtPath      string    `json:"cover_art_path" db:"cover_art_path"`
	YoutubePlaylistID string    `json:"youtube_playlist_id" db:"youtube_playlist_id"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// Song represents a song with all its metadata and processing info
type Song struct {
	ID         int       `json:"id" db:"id"`
	AlbumID    *int      `json:"album_id" db:"album_id"`
	Title      string    `json:"title" db:"title"`
	ArtistName string    `json:"artist_name" db:"artist_name"`
	Genre      string    `json:"genre" db:"genre"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`

	// Audio stems
	VocalsStemPath string `json:"vocals_stem_path" db:"vocals_stem_path"`
	MusicStemPath  string `json:"music_stem_path" db:"music_stem_path"`
	MixedAudioPath string `json:"mixed_audio_path" db:"mixed_audio_path"`
	MetadataPath   string `json:"metadata_file_path" db:"metadata_file_path"`

	// Lyrics
	Lyrics         string `json:"lyrics" db:"lyrics"`                           // Original song lyrics with [Verse], [Chorus], etc.
	LyricsKaraoke  string `json:"lyrics_karaoke,omitempty" db:"lyrics_karaoke"` // Formatted lyrics for karaoke display (no section labels)
	LyricsDisplay  string `json:"lyrics_display" db:"lyrics_display"`           // JSON
	LyricsSections string `json:"lyrics_sections" db:"lyrics_sections"`         // JSON

	// Audio analysis
	BPM             float64 `json:"bpm" db:"bpm"`
	Key             string  `json:"key" db:"key"`
	Tempo           string  `json:"tempo" db:"tempo"`
	DurationSeconds float64 `json:"duration_seconds" db:"duration_seconds"`
	VocalTiming     string  `json:"vocal_timing" db:"vocal_timing"` // JSON

	// Branding
	BrandLogoPath string `json:"brand_logo_path" db:"brand_logo_path"`
	CopyrightText string `json:"copyright_text" db:"copyright_text"`

	// Video settings
	BackgroundStyle  string  `json:"background_style" db:"background_style"`
	SpectrumStyle    string  `json:"spectrum_style" db:"spectrum_style"`     // Visualization type: showfreqs, showspectrum, showcqt, etc.
	SpectrumColor    string  `json:"spectrum_color" db:"spectrum_color"`     // Color: rainbow, cyan, blue, red, etc.
	SpectrumOpacity  float64 `json:"spectrum_opacity" db:"spectrum_opacity"` // Opacity: 0.0-1.0
	TargetResolution string  `json:"target_resolution" db:"target_resolution"`
	ShowMetadata     bool    `json:"show_metadata" db:"show_metadata"`
}

// QueueItem represents a job in the processing queue
type QueueItem struct {
	ID       int    `json:"id" db:"id"`
	SongID   int    `json:"song_id" db:"song_id"`
	Status   string `json:"status" db:"status"`
	Priority int    `json:"priority" db:"priority"`

	CurrentStep  string `json:"current_step" db:"current_step"`
	Progress     int    `json:"progress" db:"progress"`
	ErrorMessage string `json:"error_message" db:"error_message"`
	RetryCount   int    `json:"retry_count" db:"retry_count"`

	VideoFilePath string `json:"video_file_path" db:"video_file_path"`
	VideoFileSize int64  `json:"video_file_size" db:"video_file_size"`
	ThumbnailPath string `json:"thumbnail_path" db:"thumbnail_path"`

	Flag *string `json:"flag" db:"flag"` // User-reported issue: image_issue, lyrics_issue, timing_issue

	QueuedAt    time.Time  `json:"queued_at" db:"queued_at"`
	StartedAt   *time.Time `json:"started_at" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at" db:"completed_at"`
}

// YoutubeUpload represents a YouTube video upload record
type YoutubeUpload struct {
	ID                int        `json:"id" db:"id"`
	QueueID           int        `json:"queue_id" db:"queue_id"`
	SongID            int        `json:"song_id" db:"song_id"`
	YoutubeVideoID    string     `json:"youtube_video_id" db:"youtube_video_id"`
	YoutubeURL        string     `json:"youtube_url" db:"youtube_url"`
	Title             string     `json:"title" db:"title"`
	Description       string     `json:"description" db:"description"`
	Tags              string     `json:"tags" db:"tags"`
	CategoryID        int        `json:"category_id" db:"category_id"`
	PrivacyStatus     string     `json:"privacy_status" db:"privacy_status"`
	UploadStartedAt   *time.Time `json:"upload_started_at" db:"upload_started_at"`
	UploadCompletedAt *time.Time `json:"upload_completed_at" db:"upload_completed_at"`
	Views             int        `json:"views" db:"views"`
	Likes             int        `json:"likes" db:"likes"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// ProcessingLog represents a log entry for queue processing
type ProcessingLog struct {
	ID              int       `json:"id" db:"id"`
	QueueID         int       `json:"queue_id" db:"queue_id"`
	Step            string    `json:"step" db:"step"`
	Status          string    `json:"status" db:"status"`
	Message         string    `json:"message" db:"message"`
	DurationSeconds float64   `json:"duration_seconds" db:"duration_seconds"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// GeneratedImage represents an AI-generated image with its prompt
type GeneratedImage struct {
	ID             int       `json:"id" db:"id"`
	SongID         int       `json:"song_id" db:"song_id"`
	QueueID        *int      `json:"queue_id" db:"queue_id"`
	ImagePath      string    `json:"image_path" db:"image_path"`
	Prompt         string    `json:"prompt" db:"prompt"`
	NegativePrompt string    `json:"negative_prompt" db:"negative_prompt"`
	ImageType      string    `json:"image_type" db:"image_type"` // background, scene, thumbnail
	SequenceNumber *int      `json:"sequence_number" db:"sequence_number"`
	Width          int       `json:"width" db:"width"`
	Height         int       `json:"height" db:"height"`
	Model          string    `json:"model" db:"model"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// Queue status constants
const (
	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusRetrying   = "retrying"
)

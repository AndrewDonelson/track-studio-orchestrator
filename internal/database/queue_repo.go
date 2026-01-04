package database

import (
	"database/sql"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
)

// QueueRepository handles queue database operations
type QueueRepository struct {
	db *sql.DB
}

// NewQueueRepository creates a new queue repository
func NewQueueRepository(db *sql.DB) *QueueRepository {
	return &QueueRepository{db: db}
}

// GetAll returns all queue items
func (r *QueueRepository) GetAll() ([]models.QueueItem, error) {
	query := `SELECT id, song_id, status, priority,
		COALESCE(current_step, '') as current_step, 
		COALESCE(progress, 0) as progress, 
		COALESCE(error_message, '') as error_message, 
		COALESCE(retry_count, 0) as retry_count,
		COALESCE(video_file_path, '') as video_file_path, 
		COALESCE(video_file_size, 0) as video_file_size, 
		COALESCE(thumbnail_path, '') as thumbnail_path,
		queued_at, started_at, completed_at
		FROM queue ORDER BY priority DESC, queued_at ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.QueueItem
	for rows.Next() {
		var item models.QueueItem
		err := rows.Scan(
			&item.ID, &item.SongID, &item.Status, &item.Priority,
			&item.CurrentStep, &item.Progress, &item.ErrorMessage, &item.RetryCount,
			&item.VideoFilePath, &item.VideoFileSize, &item.ThumbnailPath,
			&item.QueuedAt, &item.StartedAt, &item.CompletedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// GetByID returns a queue item by ID
func (r *QueueRepository) GetByID(id int) (*models.QueueItem, error) {
	query := `SELECT id, song_id, status, priority,
		COALESCE(current_step, '') as current_step, 
		COALESCE(progress, 0) as progress, 
		COALESCE(error_message, '') as error_message, 
		COALESCE(retry_count, 0) as retry_count,
		COALESCE(video_file_path, '') as video_file_path, 
		COALESCE(video_file_size, 0) as video_file_size, 
		COALESCE(thumbnail_path, '') as thumbnail_path,
		queued_at, started_at, completed_at
		FROM queue WHERE id = ?`

	var item models.QueueItem
	err := r.db.QueryRow(query, id).Scan(
		&item.ID, &item.SongID, &item.Status, &item.Priority,
		&item.CurrentStep, &item.Progress, &item.ErrorMessage, &item.RetryCount,
		&item.VideoFilePath, &item.VideoFileSize, &item.ThumbnailPath,
		&item.QueuedAt, &item.StartedAt, &item.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &item, nil
}

// Create creates a new queue item
func (r *QueueRepository) Create(item *models.QueueItem) error {
	query := `INSERT INTO queue (song_id, status, priority)
		VALUES (?, ?, ?)`

	result, err := r.db.Exec(query, item.SongID, item.Status, item.Priority)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	item.ID = int(id)
	return nil
}

// Update updates an existing queue item
func (r *QueueRepository) Update(item *models.QueueItem) error {
	query := `UPDATE queue SET status=?, priority=?,
		current_step=?, progress=?, error_message=?, retry_count=?,
		video_file_path=?, video_file_size=?, thumbnail_path=?,
		started_at=?, completed_at=?
		WHERE id=?`

	_, err := r.db.Exec(query,
		item.Status, item.Priority,
		item.CurrentStep, item.Progress, item.ErrorMessage, item.RetryCount,
		item.VideoFilePath, item.VideoFileSize, item.ThumbnailPath,
		item.StartedAt, item.CompletedAt,
		item.ID,
	)
	return err
}

// Delete removes a queue item
func (r *QueueRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM queue WHERE id=?", id)
	return err
}

// GetNextPending returns the next pending queue item
func (r *QueueRepository) GetNextPending() (*models.QueueItem, error) {
	query := `SELECT id, song_id, status, priority,
		COALESCE(current_step, '') as current_step, 
		COALESCE(progress, 0) as progress, 
		COALESCE(error_message, '') as error_message, 
		COALESCE(retry_count, 0) as retry_count,
		COALESCE(video_file_path, '') as video_file_path, 
		COALESCE(video_file_size, 0) as video_file_size, 
		COALESCE(thumbnail_path, '') as thumbnail_path,
		queued_at, started_at, completed_at
		FROM queue 
		WHERE status = ?
		ORDER BY priority DESC, queued_at ASC
		LIMIT 1`

	var item models.QueueItem
	err := r.db.QueryRow(query, models.StatusQueued).Scan(
		&item.ID, &item.SongID, &item.Status, &item.Priority,
		&item.CurrentStep, &item.Progress, &item.ErrorMessage, &item.RetryCount,
		&item.VideoFilePath, &item.VideoFileSize, &item.ThumbnailPath,
		&item.QueuedAt, &item.StartedAt, &item.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &item, nil
}

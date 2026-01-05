package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	db *sql.DB
}

func NewDashboardHandler(db *sql.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

type DashboardStats struct {
	// Current Status
	TotalSongs      int `json:"total_songs"`
	TotalVideos     int `json:"total_videos"`
	QueuedItems     int `json:"queued_items"`
	ProcessingItems int `json:"processing_items"`
	CompletedToday  int `json:"completed_today"`
	ErrorsToday     int `json:"errors_today"`

	// Analytics
	YTDMinProcessingTime string  `json:"ytd_min_processing_time"`
	YTDMaxProcessingTime string  `json:"ytd_max_processing_time"`
	YTDAvgProcessingTime string  `json:"ytd_avg_processing_time"`
	YTDTotalVideos       int     `json:"ytd_total_videos"`
	YTDSuccessRate       float64 `json:"ytd_success_rate"`
	YTDTotalErrors       int     `json:"ytd_total_errors"`

	// Recent Activity
	RecentVideos      []RecentVideo `json:"recent_videos"`
	RecentErrors      []RecentError `json:"recent_errors"`
	GenreDistribution []GenreStats  `json:"genre_distribution"`
}

type RecentVideo struct {
	ID             int       `json:"id"`
	SongID         int       `json:"song_id"`
	Title          string    `json:"title"`
	Artist         string    `json:"artist"`
	ProcessingTime string    `json:"processing_time"`
	CompletedAt    time.Time `json:"completed_at"`
}

type RecentError struct {
	ID           int       `json:"id"`
	SongID       int       `json:"song_id"`
	Title        string    `json:"title"`
	ErrorMessage string    `json:"error_message"`
	FailedAt     time.Time `json:"failed_at"`
}

type GenreStats struct {
	Genre string `json:"genre"`
	Count int    `json:"count"`
}

func formatDuration(seconds int) string {
	if seconds < 0 {
		return "0s"
	}

	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, secs)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

func (h *DashboardHandler) GetDashboard(c *gin.Context) {
	stats := DashboardStats{}

	// Total songs
	err := h.db.QueryRow("SELECT COUNT(*) FROM songs").Scan(&stats.TotalSongs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Total completed videos from videos table
	err = h.db.QueryRow("SELECT COUNT(*) FROM videos WHERE status = 'completed'").Scan(&stats.TotalVideos)
	if err != nil {
		stats.TotalVideos = 0
	}

	// Queued items
	err = h.db.QueryRow("SELECT COUNT(*) FROM queue WHERE status = 'queued'").Scan(&stats.QueuedItems)
	if err != nil {
		stats.QueuedItems = 0
	}

	// Processing items
	err = h.db.QueryRow("SELECT COUNT(*) FROM queue WHERE status = 'processing'").Scan(&stats.ProcessingItems)
	if err != nil {
		stats.ProcessingItems = 0
	}

	// Completed today from queue (completed queue items)
	err = h.db.QueryRow("SELECT COUNT(*) FROM queue WHERE status = 'completed' AND DATE(completed_at) = DATE('now')").Scan(&stats.CompletedToday)
	if err != nil {
		stats.CompletedToday = 0
	}

	// Errors today from queue
	err = h.db.QueryRow("SELECT COUNT(*) FROM queue WHERE status = 'failed' AND DATE(completed_at) = DATE('now')").Scan(&stats.ErrorsToday)
	if err != nil {
		stats.ErrorsToday = 0
	}

	// Analytics - YTD stats
	var minSeconds, maxSeconds, totalSeconds sql.NullInt64
	var totalVideos, totalErrors sql.NullInt64

	// Calculate processing time stats from completed queue items
	err = h.db.QueryRow(`
		SELECT 
			MIN(CAST((julianday(completed_at) - julianday(started_at)) * 86400 AS INTEGER)),
			MAX(CAST((julianday(completed_at) - julianday(started_at)) * 86400 AS INTEGER)),
			AVG(CAST((julianday(completed_at) - julianday(started_at)) * 86400 AS INTEGER)),
			COUNT(*)
		FROM queue 
		WHERE status = 'completed' 
		AND started_at IS NOT NULL 
		AND completed_at IS NOT NULL
	`).Scan(&minSeconds, &maxSeconds, &totalSeconds, &totalVideos)

	if err == nil && minSeconds.Valid {
		stats.YTDMinProcessingTime = formatDuration(int(minSeconds.Int64))
		stats.YTDMaxProcessingTime = formatDuration(int(maxSeconds.Int64))
		stats.YTDAvgProcessingTime = formatDuration(int(totalSeconds.Int64))
		stats.YTDTotalVideos = int(totalVideos.Int64)
	} else {
		stats.YTDMinProcessingTime = "N/A"
		stats.YTDMaxProcessingTime = "N/A"
		stats.YTDAvgProcessingTime = "N/A"
		stats.YTDTotalVideos = 0
	}

	// Error stats from queue
	err = h.db.QueryRow("SELECT COUNT(*) FROM queue WHERE status = 'failed'").Scan(&totalErrors)
	if err == nil && totalErrors.Valid {
		stats.YTDTotalErrors = int(totalErrors.Int64)
		totalAttempts := stats.YTDTotalVideos + stats.YTDTotalErrors
		if totalAttempts > 0 {
			stats.YTDSuccessRate = float64(stats.YTDTotalVideos) / float64(totalAttempts) * 100
		} else {
			stats.YTDSuccessRate = 0.0
		}
	}

	// Recent videos from videos table (last 10)
	rows, err := h.db.Query(`
		SELECT v.id, v.song_id, s.title, s.artist_name, v.rendered_at
		FROM videos v
		JOIN songs s ON v.song_id = s.id
		WHERE v.status = 'completed'
		ORDER BY v.rendered_at DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var v RecentVideo
			var renderedAt time.Time
			err = rows.Scan(&v.ID, &v.SongID, &v.Title, &v.Artist, &renderedAt)
			if err == nil {
				v.CompletedAt = renderedAt
				v.ProcessingTime = "N/A" // Videos table doesn't track processing time
				stats.RecentVideos = append(stats.RecentVideos, v)
			}
		}
	}

	// Recent errors (last 10) from queue
	rows, err = h.db.Query(`
		SELECT q.id, q.song_id, s.title, q.error_message, q.completed_at
		FROM queue q
		JOIN songs s ON q.song_id = s.id
		WHERE q.status = 'failed'
		ORDER BY q.completed_at DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var e RecentError
			var completedAt sql.NullTime
			err = rows.Scan(&e.ID, &e.SongID, &e.Title, &e.ErrorMessage, &completedAt)
			if err == nil && completedAt.Valid {
				e.FailedAt = completedAt.Time
				stats.RecentErrors = append(stats.RecentErrors, e)
			}
		}
	}

	// Genre distribution from videos
	rows, err = h.db.Query(`
		SELECT COALESCE(v.genre, 'Unknown') as genre, COUNT(*) as count
		FROM videos v
		WHERE v.status = 'completed'
		GROUP BY v.genre
		ORDER BY count DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var g GenreStats
			err = rows.Scan(&g.Genre, &g.Count)
			if err == nil {
				stats.GenreDistribution = append(stats.GenreDistribution, g)
			}
		}
	}

	c.JSON(http.StatusOK, stats)
}

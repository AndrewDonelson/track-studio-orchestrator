package database

import (
	"database/sql"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
)

// CreateGeneratedImage inserts a new generated image record
func CreateGeneratedImage(img *models.GeneratedImage) error {
	query := `
		INSERT INTO generated_images (
			song_id, queue_id, image_path, prompt, negative_prompt,
			image_type, sequence_number, width, height, model
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := DB.Exec(query,
		img.SongID, img.QueueID, img.ImagePath, img.Prompt, img.NegativePrompt,
		img.ImageType, img.SequenceNumber, img.Width, img.Height, img.Model,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	img.ID = int(id)
	return nil
}

// CreateImagePrompt creates an image record with just a prompt (no actual image file yet)
func CreateImagePrompt(img *models.GeneratedImage) (int, error) {
	// Use the existing CreateGeneratedImage but allow empty image_path
	err := CreateGeneratedImage(img)
	if err != nil {
		return 0, err
	}
	return img.ID, nil
}

// GetImagesBySongID retrieves all images for a song
func GetImagesBySongID(songID int) ([]models.GeneratedImage, error) {
	query := `
		SELECT id, song_id, queue_id, image_path, prompt, negative_prompt,
		       image_type, sequence_number, width, height, model, created_at
		FROM generated_images
		WHERE song_id = ?
		ORDER BY image_type, sequence_number
	`
	rows, err := DB.Query(query, songID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []models.GeneratedImage
	for rows.Next() {
		var img models.GeneratedImage
		err := rows.Scan(
			&img.ID, &img.SongID, &img.QueueID, &img.ImagePath, &img.Prompt, &img.NegativePrompt,
			&img.ImageType, &img.SequenceNumber, &img.Width, &img.Height, &img.Model, &img.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, nil
}

// GetImageByID retrieves a single image by ID
func GetImageByID(id int) (*models.GeneratedImage, error) {
	query := `
		SELECT id, song_id, queue_id, image_path, prompt, negative_prompt,
		       image_type, sequence_number, width, height, model, created_at
		FROM generated_images
		WHERE id = ?
	`
	var img models.GeneratedImage
	err := DB.QueryRow(query, id).Scan(
		&img.ID, &img.SongID, &img.QueueID, &img.ImagePath, &img.Prompt, &img.NegativePrompt,
		&img.ImageType, &img.SequenceNumber, &img.Width, &img.Height, &img.Model, &img.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &img, nil
}

// UpdateImagePrompt updates the prompt and negative prompt for an image
func UpdateImagePrompt(id int, prompt, negativePrompt string) error {
	query := `
		UPDATE generated_images
		SET prompt = ?, negative_prompt = ?
		WHERE id = ?
	`
	_, err := DB.Exec(query, prompt, negativePrompt, id)
	return err
}

// UpdateImagePath updates the image_path for a generated image
func UpdateImagePath(id int, imagePath string) error {
	query := `
		UPDATE generated_images
		SET image_path = ?
		WHERE id = ?
	`
	_, err := DB.Exec(query, imagePath, id)
	return err
}

// DeleteImagesBySongID deletes all images for a song
func DeleteImagesBySongID(songID int) error {
	query := `DELETE FROM generated_images WHERE song_id = ?`
	_, err := DB.Exec(query, songID)
	return err
}

// DeleteImagesByQueueID deletes all images for a queue item
func DeleteImagesByQueueID(queueID int) error {
	query := `DELETE FROM generated_images WHERE queue_id = ?`
	_, err := DB.Exec(query, queueID)
	return err
}

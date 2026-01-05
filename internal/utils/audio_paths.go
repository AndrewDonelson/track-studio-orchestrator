package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetSongAudioDir returns the audio directory for a specific song
func GetSongAudioDir(songID int) string {
	return filepath.Join(GetAudioPath(), fmt.Sprintf("song_%d", songID))
}

// GetSongVocalPath returns the path to the vocal stem for a song
// Returns empty string if file doesn't exist
func GetSongVocalPath(songID int) string {
	dir := GetSongAudioDir(songID)

	// Try common extensions
	extensions := []string{".wav", ".mp3", ".flac", ".m4a"}
	for _, ext := range extensions {
		path := filepath.Join(dir, "vocal"+ext)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// GetSongMusicPath returns the path to the music/instrumental stem for a song
// Returns empty string if file doesn't exist
func GetSongMusicPath(songID int) string {
	dir := GetSongAudioDir(songID)

	// Try common extensions
	extensions := []string{".wav", ".mp3", ".flac", ".m4a"}
	for _, ext := range extensions {
		path := filepath.Join(dir, "music"+ext)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// GetSongMixedPath returns the path to the mixed audio for a song
// Returns empty string if file doesn't exist
func GetSongMixedPath(songID int) string {
	dir := GetSongAudioDir(songID)

	// Try common extensions
	extensions := []string{".wav", ".mp3", ".flac", ".m4a"}
	for _, ext := range extensions {
		path := filepath.Join(dir, "mixed"+ext)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// GetSongAudioPath returns the best available audio file for a song
// Priority: music stem > vocal stem > mixed audio
// Returns empty string if no audio files exist
func GetSongAudioPath(songID int) string {
	if path := GetSongMusicPath(songID); path != "" {
		return path
	}
	if path := GetSongVocalPath(songID); path != "" {
		return path
	}
	if path := GetSongMixedPath(songID); path != "" {
		return path
	}
	return ""
}

// HasSongAudio checks if a song has any audio files
func HasSongAudio(songID int) bool {
	return GetSongAudioPath(songID) != ""
}

// HasSongStems checks if a song has both vocal and music stems
func HasSongStems(songID int) bool {
	return GetSongVocalPath(songID) != "" && GetSongMusicPath(songID) != ""
}

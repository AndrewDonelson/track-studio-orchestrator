-- Migration: Add whisper_engine field to songs table
-- Created: 2026-01-05
-- Purpose: Track which Whisper engine was used for karaoke generation (WhisperX GPU or Faster-Whisper CPU)

-- Add whisper_engine column if it doesn't exist
ALTER TABLE songs ADD COLUMN IF NOT EXISTS whisper_engine TEXT DEFAULT '';

-- Create index for faster queries filtering by engine
CREATE INDEX IF NOT EXISTS idx_songs_whisper_engine ON songs(whisper_engine);

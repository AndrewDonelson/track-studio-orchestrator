-- Migration: Add metadata fields to videos table for filtering and display
-- Adds genre, BPM, key, tempo, and flag fields from songs table

ALTER TABLE videos ADD COLUMN genre TEXT;
ALTER TABLE videos ADD COLUMN bpm REAL;
ALTER TABLE videos ADD COLUMN key TEXT;
ALTER TABLE videos ADD COLUMN tempo TEXT;
ALTER TABLE videos ADD COLUMN flag TEXT;  -- User-reported issues: 'image_issue', 'lyrics_issue', 'timing_issue', or NULL

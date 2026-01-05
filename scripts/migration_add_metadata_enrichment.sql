-- Migration: Add metadata enrichment fields to songs table
-- Date: 2026-01-05
-- Purpose: Enable AI-powered genre classification, tagging, and metadata enrichment

-- Add metadata enrichment columns
ALTER TABLE songs ADD COLUMN genre_primary TEXT;
ALTER TABLE songs ADD COLUMN genre_secondary TEXT; -- JSON array
ALTER TABLE songs ADD COLUMN tags TEXT; -- JSON array
ALTER TABLE songs ADD COLUMN style_descriptors TEXT; -- JSON array
ALTER TABLE songs ADD COLUMN mood TEXT; -- JSON array
ALTER TABLE songs ADD COLUMN themes TEXT; -- JSON array
ALTER TABLE songs ADD COLUMN similar_artists TEXT; -- JSON array
ALTER TABLE songs ADD COLUMN summary TEXT;
ALTER TABLE songs ADD COLUMN target_audience TEXT;
ALTER TABLE songs ADD COLUMN energy_level TEXT;
ALTER TABLE songs ADD COLUMN vocal_style TEXT;
ALTER TABLE songs ADD COLUMN metadata_enriched_at TIMESTAMP;
ALTER TABLE songs ADD COLUMN metadata_version INTEGER DEFAULT 1;

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_songs_genre_primary ON songs(genre_primary);
CREATE INDEX IF NOT EXISTS idx_songs_energy_level ON songs(energy_level);
CREATE INDEX IF NOT EXISTS idx_songs_metadata_enriched ON songs(metadata_enriched_at);

-- Note: SQLite doesn't support ENUM types or GIN indexes like PostgreSQL
-- We'll handle genre validation in the application layer

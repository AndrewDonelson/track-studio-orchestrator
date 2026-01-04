-- Migration: Add karaoke customization fields to songs table

ALTER TABLE songs ADD COLUMN karaoke_font_family TEXT DEFAULT 'Arial';
ALTER TABLE songs ADD COLUMN karaoke_font_size INTEGER DEFAULT 96;
ALTER TABLE songs ADD COLUMN karaoke_primary_color TEXT DEFAULT '4169E1';
ALTER TABLE songs ADD COLUMN karaoke_primary_border_color TEXT DEFAULT 'FFFFFF';
ALTER TABLE songs ADD COLUMN karaoke_highlight_color TEXT DEFAULT 'FFD700';
ALTER TABLE songs ADD COLUMN karaoke_highlight_border_color TEXT DEFAULT 'FFFFFF';
ALTER TABLE songs ADD COLUMN karaoke_alignment INTEGER DEFAULT 5;
ALTER TABLE songs ADD COLUMN karaoke_margin_bottom INTEGER DEFAULT 0;

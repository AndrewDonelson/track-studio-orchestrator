-- Add settings table
CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    master_prompt TEXT DEFAULT '',
    master_negative_prompt TEXT DEFAULT '',
    brand_logo_path TEXT DEFAULT '',
    data_storage_path TEXT DEFAULT '~/track-studio-data',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default settings if not exists
INSERT OR IGNORE INTO settings (id, data_storage_path) VALUES (1, '~/track-studio-data');

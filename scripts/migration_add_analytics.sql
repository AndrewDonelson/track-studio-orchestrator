-- Analytics table for tracking video generation statistics

CREATE TABLE IF NOT EXISTS analytics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_name TEXT NOT NULL UNIQUE,
    metric_value REAL NOT NULL,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Seed initial analytics metrics
INSERT OR IGNORE INTO analytics (metric_name, metric_value) VALUES 
    ('ytd_min_processing_time', 999999),
    ('ytd_max_processing_time', 0),
    ('ytd_avg_processing_time', 0),
    ('ytd_total_videos_generated', 0),
    ('ytd_total_processing_time', 0),
    ('ytd_success_rate', 100.0),
    ('ytd_total_errors', 0),
    ('ytd_total_retries', 0);

-- Create index for faster metric lookups
CREATE INDEX IF NOT EXISTS idx_analytics_metric_name ON analytics(metric_name);

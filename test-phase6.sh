#!/bin/bash
# Test Phase 6: Lyrics Processing & Timing Alignment

cd /home/andrew/Development/Fullstack-Projects/TrackStudio/track-studio-orchestrator

echo "🧪 Testing Phase 6: Lyrics Processing"
echo "======================================"
echo

# Start server in background
echo "Starting orchestrator..."
bin/orchestrator > /tmp/orchestrator-phase6.log 2>&1 &
SERVER_PID=$!
sleep 3

# Add queue item
echo "Adding 'Land of Love' to queue..."
QUEUE_RESULT=$(curl -s -X POST http://localhost:8080/api/v1/queue \
  -H "Content-Type: application/json" \
  -d '{"song_id": 1, "priority": 1}')

echo "$QUEUE_RESULT" | jq '{id, status, song_id}'

# Wait for processing (audio analysis + lyrics takes time)
echo
echo "⏳ Waiting for audio analysis and lyrics processing..."
echo "   This may take 30-60 seconds for WAV file analysis..."
sleep 45

# Check results
echo
echo "📊 Song Analysis Results:"
echo "========================="
sqlite3 data/trackstudio.db << 'SQL'
.mode column
.headers on
SELECT 
  id,
  title,
  ROUND(bpm, 1) as bpm,
  key,
  tempo,
  ROUND(duration_seconds, 1) as duration,
  CASE WHEN lyrics_sections IS NOT NULL THEN 'Yes' ELSE 'No' END as has_sections,
  CASE WHEN lyrics_display IS NOT NULL THEN 'Yes' ELSE 'No' END as has_timing
FROM songs WHERE id = 1;
SQL

echo
echo "📝 Lyrics Sections:"
echo "==================="
sqlite3 data/trackstudio.db "SELECT lyrics_sections FROM songs WHERE id = 1" | jq '.[] | {type, number, line_count: (.lines | length)}'

echo
echo "⏱️  Timed Lines Sample (first 3):"
echo "=================================="
sqlite3 data/trackstudio.db "SELECT lyrics_display FROM songs WHERE id = 1" | jq '.[0:3] | .[] | {line, start: (.start_time | tonumber | round), end: (.end_time | tonumber | round)}'

echo
echo "📋 Queue Status:"
echo "================"
sqlite3 data/trackstudio.db << 'SQL'
.mode column
.headers on
SELECT 
  id,
  status,
  progress,
  current_step,
  CASE WHEN error_message IS NULL THEN 'None' ELSE error_message END as error
FROM queue ORDER BY id DESC LIMIT 1;
SQL

# Show processing log
echo
echo "📝 Processing Log (last 30 lines):"
echo "==================================="
tail -30 /tmp/orchestrator-phase6.log | grep -E "(Audio analysis|Lyrics|vocal segments|sections|lines)"

# Cleanup
kill $SERVER_PID 2>/dev/null
echo
echo "✅ Phase 6 Test Complete!"

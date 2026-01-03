#!/bin/bash
# Test Phase 5: Audio Analysis Integration

cd /home/andrew/Development/Fullstack-Projects/TrackStudio/track-studio-orchestrator

echo "🧪 Testing Phase 5: Audio Analysis"
echo "=================================="
echo

# Start server in background
echo "Starting orchestrator..."
bin/orchestrator > /tmp/orchestrator-test.log 2>&1 &
SERVER_PID=$!
sleep 3

# Add queue item
echo "Adding song to queue..."
curl -s -X POST http://localhost:8080/api/v1/queue \
  -H "Content-Type: application/json" \
  -d '{"song_id": 1, "priority": 1}' | jq '.id, .status'

# Wait for processing
echo "Waiting for audio analysis to complete..."
sleep 15

# Check results
echo
echo "📊 Audio Analysis Results:"
echo "=========================="
sqlite3 data/trackstudio.db << 'SQL'
SELECT 
  'Song: ' || title as info,
  'BPM: ' || COALESCE(CAST(bpm AS TEXT), 'NULL') as bpm_info,
  'Key: ' || COALESCE(key, 'NULL') as key_info,
  'Tempo: ' || COALESCE(tempo, 'NULL') as tempo_info,
  'Duration: ' || COALESCE(CAST(duration_seconds AS TEXT), 'NULL') || 's' as duration_info
FROM songs WHERE id = 1;
SQL

echo
echo "📋 Queue Status:"
echo "================"
sqlite3 data/trackstudio.db "SELECT id, status, progress, current_step FROM queue ORDER BY id DESC LIMIT 1"

# Show last few log lines
echo
echo "📝 Last Log Lines:"
echo "=================="
tail -20 /tmp/orchestrator-test.log

# Cleanup
kill $SERVER_PID 2>/dev/null
echo
echo "✅ Test complete!"

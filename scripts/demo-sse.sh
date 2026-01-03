#!/bin/bash
# Test SSE with actual queue updates

echo "Starting SSE listener in background..."
timeout 15 curl -N -H "Accept: text/event-stream" http://localhost:8080/api/v1/progress/stream 2>/dev/null &
CURL_PID=$!

sleep 2

echo "Adding item to queue..."
curl -s -X POST http://localhost:8080/api/v1/queue \
  -H "Content-Type: application/json" \
  -d '{"song_id":1,"priority":10}' | jq -r '.id' > /tmp/queue_id.txt

QUEUE_ID=$(cat /tmp/queue_id.txt)
echo "Created queue item: $QUEUE_ID"
sleep 1

echo "Updating queue item progress to 25%..."
curl -s -X PUT http://localhost:8080/api/v1/queue/$QUEUE_ID \
  -H "Content-Type: application/json" \
  -d "{\"status\":\"processing\",\"current_step\":\"Analyzing audio\",\"progress\":25,\"song_id\":1,\"priority\":10}" > /dev/null
sleep 1

echo "Updating queue item progress to 50%..."
curl -s -X PUT http://localhost:8080/api/v1/queue/$QUEUE_ID \
  -H "Content-Type: application/json" \
  -d "{\"status\":\"processing\",\"current_step\":\"Generating images\",\"progress\":50,\"song_id\":1,\"priority\":10}" > /dev/null
sleep 1

echo "Updating queue item progress to 75%..."
curl -s -X PUT http://localhost:8080/api/v1/queue/$QUEUE_ID \
  -H "Content-Type: application/json" \
  -d "{\"status\":\"processing\",\"current_step\":\"Rendering video\",\"progress\":75,\"song_id\":1,\"priority\":10}" > /dev/null
sleep 1

echo "Completing queue item..."
curl -s -X PUT http://localhost:8080/api/v1/queue/$QUEUE_ID \
  -H "Content-Type: application/json" \
  -d "{\"status\":\"completed\",\"current_step\":\"Done\",\"progress\":100,\"song_id\":1,\"priority\":10}" > /dev/null

echo ""
echo "Waiting for SSE stream to complete..."
wait $CURL_PID

rm -f /tmp/queue_id.txt
echo "Test complete!"

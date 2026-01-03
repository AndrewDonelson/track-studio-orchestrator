#!/bin/bash
# API Test Script for Track Studio Orchestrator

BASE_URL="http://localhost:8080"

echo "=== Track Studio Orchestrator API Tests ==="
echo ""

# Health Check
echo "1. Health Check"
curl -s "$BASE_URL/health" | jq .
echo ""

# Get All Songs
echo "2. Get All Songs"
curl -s "$BASE_URL/api/v1/songs" | jq .
echo ""

# Get Song by ID
echo "3. Get Song by ID (ID=1)"
curl -s "$BASE_URL/api/v1/songs/1" | jq .
echo ""

# Get Progress Stats
echo "4. Get Progress Monitoring Stats"
curl -s "$BASE_URL/api/v1/progress/stats" | jq .
echo ""

# Add Song to Queue
echo "5. Add Song to Queue (will broadcast SSE update)"
curl -s -X POST "$BASE_URL/api/v1/queue" \
  -H "Content-Type: application/json" \
  -d '{"song_id":1,"priority":10}' | jq .
echo ""

# Get Queue
echo "6. Get Queue"
curl -s "$BASE_URL/api/v1/queue" | jq .
echo ""

# Get Next Queue Item
echo "7. Get Next Pending Queue Item"
curl -s "$BASE_URL/api/v1/queue/next" | jq .
echo ""

# Update Queue Item (will broadcast SSE update)
echo "8. Update Queue Item (simulating progress update)"
QUEUE_ID=$(curl -s "$BASE_URL/api/v1/queue" | jq -r '.queue[0].id')
if [ "$QUEUE_ID" != "null" ]; then
  curl -s -X PUT "$BASE_URL/api/v1/queue/$QUEUE_ID" \
    -H "Content-Type: application/json" \
    -d "{\"status\":\"processing\",\"current_step\":\"Testing API\",\"progress\":75,\"song_id\":1,\"priority\":10}" | jq .
  echo ""
fi

echo "=== All Tests Complete ==="
echo ""
echo "💡 To test live progress monitoring, run in another terminal:"
echo "   ./scripts/test-sse.sh"

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

# Add Song to Queue
echo "4. Add Song to Queue"
curl -s -X POST "$BASE_URL/api/v1/queue" \
  -H "Content-Type: application/json" \
  -d '{"song_id":1,"priority":10}' | jq .
echo ""

# Get Queue
echo "5. Get Queue"
curl -s "$BASE_URL/api/v1/queue" | jq .
echo ""

# Get Next Queue Item
echo "6. Get Next Pending Queue Item"
curl -s "$BASE_URL/api/v1/queue/next" | jq .
echo ""

# Get Queue Item by ID
echo "7. Get Queue Item by ID (ID=1)"
curl -s "$BASE_URL/api/v1/queue/1" | jq .
echo ""

echo "=== All Tests Complete ==="

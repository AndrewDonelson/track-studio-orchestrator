#!/bin/bash
# Test SSE (Server-Sent Events) for live progress monitoring

echo "=== Testing Live Progress Monitoring ==="
echo ""
echo "This will connect to the progress stream and display live updates."
echo "In another terminal, add items to the queue or update queue items to see live updates."
echo ""
echo "Press Ctrl+C to stop."
echo ""

# Start listening to the progress stream
curl -N -H "Accept: text/event-stream" http://localhost:8080/api/v1/progress/stream

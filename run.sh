#!/bin/bash

# Find and kill any running trackstudio-server processes
PID=$(pgrep -f './bin/trackstudio-server')
if [ -n "$PID" ]; then
  echo "Killing running trackstudio-server (PID: $PID)"
  kill $PID
  sleep 1
  # If still running, force kill
  if ps -p $PID > /dev/null; then
    echo "Force killing trackstudio-server (PID: $PID)"
    kill -9 $PID
  fi
else
  echo "No running trackstudio-server found."
fi

# Start the server
nohup ./bin/trackstudio-server > trackstudio-server.log 2>&1 &
NEW_PID=$!
echo "Started trackstudio-server (PID: $NEW_PID)"

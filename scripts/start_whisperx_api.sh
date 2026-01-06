#!/bin/bash

# WhisperX API Service Startup Script for CQAI
# Runs the FastAPI service for WhisperX transcription

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_SCRIPT="$SCRIPT_DIR/whisperx_api.py"
REQUIREMENTS="$SCRIPT_DIR/requirements.txt"
HOST="0.0.0.0"
PORT="8181"
LOG_FILE="$SCRIPT_DIR/whisperx_api.log"

# Function to check if port is in use
check_port() {
    if lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null ; then
        echo "Port $PORT is already in use. Please stop the existing service or choose a different port."
        exit 1
    fi
}

# Function to install dependencies
install_dependencies() {
    echo "Installing Python dependencies..."
    if command -v pip3 &> /dev/null; then
        pip3 install -r "$REQUIREMENTS"
    elif command -v pip &> /dev/null; then
        pip install -r "$REQUIREMENTS"
    else
        echo "Error: pip not found. Please install pip first."
        exit 1
    fi
}

# Function to start the service
start_service() {
    echo "Starting WhisperX API service on $HOST:$PORT"
    echo "Log file: $LOG_FILE"

    # Check if Python 3 is available
    if ! command -v python3 &> /dev/null; then
        echo "Error: python3 not found. Please install Python 3."
        exit 1
    fi

    # Check for virtual environment
    VENV_DIR="$SCRIPT_DIR/whisperx_env"
    if [[ -d "$VENV_DIR" ]]; then
        echo "Activating virtual environment: $VENV_DIR"
        source "$VENV_DIR/bin/activate"
        PYTHON_CMD="$VENV_DIR/bin/python3"
        PIP_CMD="$VENV_DIR/bin/pip"
    else
        echo "No virtual environment found, using system Python"
        PYTHON_CMD="python3"
        PIP_CMD="pip3"
    fi

    # Install dependencies if requirements.txt exists and not in venv
    if [[ -f "$REQUIREMENTS" && ! -d "$VENV_DIR" ]]; then
        install_dependencies
    fi

    # Check if API script exists
    if [[ ! -f "$API_SCRIPT" ]]; then
        echo "Error: API script not found at $API_SCRIPT"
        exit 1
    fi

    # Start the service in background
    nohup "$PYTHON_CMD" "$API_SCRIPT" > "$LOG_FILE" 2>&1 &
    SERVICE_PID=$!

    echo "Service started with PID: $SERVICE_PID"
    echo "API available at: http://$HOST:$PORT"
    echo "Health check: http://$HOST:$PORT/health"

    # Save PID for later stopping
    echo $SERVICE_PID > "$SCRIPT_DIR/whisperx_api.pid"

    # Wait a moment and check if service is running
    sleep 2
    if kill -0 $SERVICE_PID 2>/dev/null; then
        echo "Service is running successfully"
    else
        echo "Error: Service failed to start. Check log file: $LOG_FILE"
        exit 1
    fi
}

# Function to stop the service
stop_service() {
    PID_FILE="$SCRIPT_DIR/whisperx_api.pid"

    if [[ -f "$PID_FILE" ]]; then
        SERVICE_PID=$(cat "$PID_FILE")
        if kill -0 $SERVICE_PID 2>/dev/null; then
            echo "Stopping service with PID: $SERVICE_PID"
            kill $SERVICE_PID
            sleep 2
            if kill -0 $SERVICE_PID 2>/dev/null; then
                echo "Force killing service..."
                kill -9 $SERVICE_PID
            fi
            echo "Service stopped"
        else
            echo "Service is not running"
        fi
        rm -f "$PID_FILE"
    else
        echo "PID file not found. Service may not be running."
    fi
}

# Function to check service status
status_service() {
    PID_FILE="$SCRIPT_DIR/whisperx_api.pid"

    if [[ -f "$PID_FILE" ]]; then
        SERVICE_PID=$(cat "$PID_FILE")
        if kill -0 $SERVICE_PID 2>/dev/null; then
            echo "Service is running with PID: $SERVICE_PID"
            echo "API available at: http://$HOST:$PORT"
        else
            echo "Service is not running (stale PID file)"
            rm -f "$PID_FILE"
        fi
    else
        echo "Service is not running"
    fi
}

# Function to show logs
show_logs() {
    if [[ -f "$LOG_FILE" ]]; then
        tail -f "$LOG_FILE"
    else
        echo "Log file not found: $LOG_FILE"
    fi
}

# Main script logic
case "${1:-start}" in
    start)
        check_port
        start_service
        ;;
    stop)
        stop_service
        ;;
    restart)
        stop_service
        sleep 2
        check_port
        start_service
        ;;
    status)
        status_service
        ;;
    logs)
        show_logs
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|logs}"
        echo ""
        echo "Commands:"
        echo "  start   - Start the WhisperX API service"
        echo "  stop    - Stop the WhisperX API service"
        echo "  restart - Restart the WhisperX API service"
        echo "  status  - Check service status"
        echo "  logs    - Show service logs"
        exit 1
        ;;
esac
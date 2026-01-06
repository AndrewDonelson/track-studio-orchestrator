#!/bin/bash

# CQAI Deployment Script for WhisperX API Service
# Deploys the WhisperX API to cqai.nlaakstudios (192.168.1.76)

set -e

# Configuration
CQAI_HOST="192.168.1.76"
CQAI_USER="andrew"  # Adjust as needed
PROJECT_DIR="/home/andrew/Development/Fullstack-Projects/TrackStudio"
ORCHESTRATOR_DIR="$PROJECT_DIR/track-studio-orchestrator"
SCRIPTS_DIR="$ORCHESTRATOR_DIR/scripts"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

echo_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if we're on CQAI
check_environment() {
    local current_host=$(hostname -I | awk '{print $1}')
    if [[ "$current_host" != "192.168.1.76" ]]; then
        echo_warn "Not running on CQAI ($CQAI_HOST). Current host: $current_host"
        echo_info "This script should be run on CQAI or files will be copied via SSH"
    fi
}

# Function to copy files to CQAI if not already there
copy_to_cqai() {
    if [[ "$(hostname -I | awk '{print $1}')" != "$CQAI_HOST" ]]; then
        echo_info "Copying WhisperX API files to CQAI..."

        # Create remote directory
        ssh "$CQAI_USER@$CQAI_HOST" "mkdir -p ~/whisperx-api"

        # Copy files
        scp "$SCRIPTS_DIR/whisperx_api.py" "$CQAI_USER@$CQAI_HOST:~/whisperx-api/"
        scp "$SCRIPTS_DIR/requirements.txt" "$CQAI_USER@$CQAI_HOST:~/whisperx-api/"
        scp "$SCRIPTS_DIR/start_whisperx_api.sh" "$CQAI_USER@$CQAI_HOST:~/whisperx-api/"
        scp "$SCRIPTS_DIR/WHISPERX_API_README.md" "$CQAI_USER@$CQAI_HOST:~/whisperx-api/"
        scp "$SCRIPTS_DIR/run_whisperx_docker.sh" "$CQAI_USER@$CQAI_HOST:~/whisperx-api/"

        echo_info "Files copied to CQAI"
    fi
}

# Function to install dependencies on CQAI
install_dependencies() {
    local target_host="$1"

    if [[ -n "$target_host" ]]; then
        echo_info "Checking Python environment on $target_host..."
        ssh "$CQAI_USER@$target_host" "
            cd ~/whisperx-api
            # Check if required packages are already installed
            if python3 -c 'import fastapi, uvicorn, pydantic' 2>/dev/null; then
                echo 'Python dependencies already installed'
            else
                echo 'Setting up Python virtual environment...'
                # Create virtual environment
                python3 -m venv whisperx_env
                # Activate and install
                source whisperx_env/bin/activate
                pip install --upgrade pip
                pip install -r requirements.txt
                echo 'Virtual environment created at ~/whisperx-api/whisperx_env'
            fi
        "
    else
        echo_info "Checking Python environment locally..."
        cd "$SCRIPTS_DIR"
        if python3 -c 'import fastapi, uvicorn, pydantic' 2>/dev/null; then
            echo_info "Python dependencies already installed"
        else
            echo_info "Setting up Python virtual environment..."
            # Create virtual environment
            python3 -m venv whisperx_env
            # Activate and install
            source whisperx_env/bin/activate
            pip install --upgrade pip
            pip install -r requirements.txt
            echo_info "Virtual environment created at $SCRIPTS_DIR/whisperx_env"
        fi
    fi
}

# Function to start the service
start_service() {
    local target_host="$1"

    if [[ -n "$target_host" ]]; then
        echo_info "Starting WhisperX API service on $target_host..."
        ssh "$CQAI_USER@$target_host" "
            cd ~/whisperx-api
            chmod +x start_whisperx_api.sh
            ./start_whisperx_api.sh start
        "
    else
        echo_info "Starting WhisperX API service locally..."
        cd "$SCRIPTS_DIR"
        chmod +x start_whisperx_api.sh
        ./start_whisperx_api.sh start
    fi
}

# Function to check service status
check_service() {
    local target_host="$1"

    if [[ -n "$target_host" ]]; then
        echo_info "Checking service status on $target_host..."
        ssh "$CQAI_USER@$target_host" "
            cd ~/whisperx-api
            ./start_whisperx_api.sh status
        "
    else
        echo_info "Checking service status locally..."
        cd "$SCRIPTS_DIR"
        ./start_whisperx_api.sh status
    fi
}

# Function to test the API
test_api() {
    local target_host="$1"

    if [[ -n "$target_host" ]]; then
        echo_info "Testing API on $target_host..."
        ssh "$CQAI_USER@$target_host" "
            curl -s http://localhost:8181/health | python3 -m json.tool
        "
    else
        echo_info "Testing API locally..."
        curl -s http://localhost:8181/health | python3 -m json.tool
    fi
}

# Main deployment logic
main() {
    local deploy_remote=false
    local skip_start=false

    echo_info "CQAI WhisperX API Deployment Script"
    echo_info "This script checks for existing Python packages before installing new ones"

    while [[ $# -gt 0 ]]; do
        case $1 in
            --remote)
                deploy_remote=true
                shift
                ;;
            --no-start)
                skip_start=true
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Deploy WhisperX API service to CQAI"
                echo ""
                echo "OPTIONS:"
                echo "  --remote     Deploy to remote CQAI host"
                echo "  --no-start   Skip starting the service"
                echo "  --help       Show this help"
                exit 0
                ;;
            *)
                echo_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    check_environment

    if [[ "$deploy_remote" == true ]]; then
        copy_to_cqai
        install_dependencies "$CQAI_HOST"
        if [[ "$skip_start" != true ]]; then
            start_service "$CQAI_HOST"
            sleep 3
            check_service "$CQAI_HOST"
            test_api "$CQAI_HOST"
        fi
    else
        install_dependencies ""
        if [[ "$skip_start" != true ]]; then
            start_service ""
            sleep 3
            check_service ""
            test_api ""
        fi
    fi

    echo_info "Deployment completed!"
    echo_info "API will be available at: http://$CQAI_HOST:8181"
    echo_info "Health check: http://$CQAI_HOST:8181/health"
}

main "$@"
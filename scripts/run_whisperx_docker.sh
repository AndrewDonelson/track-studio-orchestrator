#!/bin/bash

# WhisperX Docker Startup Script for CQAI
# This script runs WhisperX as a Docker container with GPU support

set -e

# Configuration
IMAGE_NAME="emsi/whisperx:latest"
MODEL_CACHE_DIR="${HOME}/.cache/huggingface/hub"
INPUT_DIR="/tmp/whisper_input"
OUTPUT_DIR="/tmp/whisper_output"

# Create directories if they don't exist
mkdir -p "$INPUT_DIR"
mkdir -p "$OUTPUT_DIR"
mkdir -p "$MODEL_CACHE_DIR"

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS] INPUT_FILE [LYRICS_FILE]"
    echo ""
    echo "Run WhisperX as a Docker container on CQAI"
    echo ""
    echo "OPTIONS:"
    echo "  -h, --help              Show this help message"
    echo "  -l, --language LANG     Language code (default: en)"
    echo "  -m, --model MODEL       Model to use (default: large-v2)"
    echo "  -o, --output DIR        Output directory (default: /tmp/whisper_output)"
    echo "  -i, --input DIR         Input directory (default: /tmp/whisper_input)"
    echo "  --cpu                   Use CPU version instead of GPU"
    echo "  --align                 Force align to provided lyrics (requires LYRICS_FILE)"
    echo "  --gpu-device N          Set the CUDA GPU device to use (e.g., 0, 1, 2)"
    echo ""
    echo "INPUT_FILE: Path to the audio/video file to process"
    echo "LYRICS_FILE: Path to text file with lyrics (for forced alignment)"
    echo ""
    echo "Examples:"
    echo "  $0 audio.mp3                    # Basic transcription (all GPUs)"
    echo "  $0 --gpu-device 1 audio.mp3     # Use only GPU 1 for processing"
    echo "  $0 --align audio.mp3 lyrics.txt # Forced alignment with lyrics"
    echo "  $0 -l es --model medium audio.mp3"
    echo "  $0 --cpu audio.wav"
}

# Default values
LANGUAGE="en"
MODEL="large-v2"
USE_GPU=true
ALIGN_MODE=false
LYRICS_FILE=""

# Parse command line arguments
GPU_DEVICE=""  # Default: use all available GPUs
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -l|--language)
            LANGUAGE="$2"
            shift 2
            ;;
        -m|--model)
            MODEL="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -i|--input)
            INPUT_DIR="$2"
            shift 2
            ;;
        --cpu)
            USE_GPU=false
            shift
            ;;
        --align)
            ALIGN_MODE=true
            shift
            ;;
        --gpu-device)
            GPU_DEVICE="$2"
            shift 2
            ;;
        -* )
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            if [[ -z "$INPUT_FILE" ]]; then
                INPUT_FILE="$1"
            elif [[ -z "$LYRICS_FILE" ]]; then
                LYRICS_FILE="$1"
            else
                echo "Too many arguments"
                usage
                exit 1
            fi
            shift
            ;;
    esac
done

# Check if input file is provided
if [[ -z "$INPUT_FILE" ]]; then
    echo "Error: No input file specified"
    usage
    exit 1
fi

# Check if input file exists
if [[ ! -f "$INPUT_FILE" ]]; then
    echo "Error: Input file '$INPUT_FILE' does not exist"
    exit 1
fi

# Check if lyrics file is required for alignment
if [[ "$ALIGN_MODE" == "true" && -z "$LYRICS_FILE" ]]; then
    echo "Error: --align requires a lyrics file"
    usage
    exit 1
fi

# Check if lyrics file exists when provided
if [[ -n "$LYRICS_FILE" && ! -f "$LYRICS_FILE" ]]; then
    echo "Error: Lyrics file '$LYRICS_FILE' does not exist"
    exit 1
fi

# Determine image to use
if [[ "$USE_GPU" == "true" ]]; then
    IMAGE_NAME="emsi/whisperx:latest"
    if [[ -n "$GPU_DEVICE" ]]; then
        GPU_FLAG="--gpus device=$GPU_DEVICE"
        echo "Using GPU-enabled WhisperX container (GPU device: $GPU_DEVICE)"
    else
        GPU_FLAG="--gpus all"
        echo "Using GPU-enabled WhisperX container (all GPUs)"
    fi
else
    IMAGE_NAME="emsi/whisperx:latest"
    GPU_FLAG=""
    echo "Using CPU-only WhisperX container"
fi

# Copy input file to input directory
INPUT_FILENAME=$(basename "$INPUT_FILE")
cp "$INPUT_FILE" "$INPUT_DIR/"

# Read lyrics if provided
LYRICS_TEXT=""
if [[ -n "$LYRICS_FILE" ]]; then
    LYRICS_TEXT=$(cat "$LYRICS_FILE" | tr '\n' ' ' | sed 's/  */ /g')
fi

echo "Starting WhisperX Docker container..."
echo "Input file: $INPUT_FILENAME"
echo "Language: $LANGUAGE"
echo "Model: $MODEL"
echo "Output directory: $OUTPUT_DIR"
if [[ "$ALIGN_MODE" == "true" ]]; then
    echo "Mode: Forced alignment with lyrics"
else
    echo "Mode: Transcription"
fi

# Build the command
CMD="whisperx \"/input/$INPUT_FILENAME\" --language \"$LANGUAGE\" --model \"$MODEL\" --output_dir \"/output\" --output_format srt,vtt,txt,json"

if [[ "$ALIGN_MODE" == "true" && -n "$LYRICS_TEXT" ]]; then
    CMD="$CMD --text \"$LYRICS_TEXT\""
fi

# Run the Docker container
docker run --rm \
    $GPU_FLAG \
    -v "$MODEL_CACHE_DIR:/root/.cache/huggingface/hub" \
    -v "$INPUT_DIR:/input" \
    -v "$OUTPUT_DIR:/output" \
    "$IMAGE_NAME" \
    bash -c "$CMD"

echo "Transcription completed. Output files are in: $OUTPUT_DIR"

# Usage info for new option
#   --gpu-device N   Set the CUDA GPU device to use (e.g., 0, 1, 2)
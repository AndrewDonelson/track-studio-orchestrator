# WhisperX API Service

A REST API service that wraps WhisperX Docker container for audio transcription and timing extraction, designed to run on CQAI infrastructure.

## Features

- **Audio Transcription**: Convert speech to text with precise word-level timestamps
- **Multiple Formats**: Returns results in SRT, VTT, TXT, and JSON formats
- **Forced Alignment**: Optional alignment mode for better accuracy with known lyrics
- **Asynchronous Processing**: Background job processing for large files
- **REST API**: Clean HTTP endpoints for integration

## Quick Deployment on CQAI

```bash
# From your local machine (copies files to CQAI and deploys)
./deploy_whisperx_api.sh --remote

# Or run directly on CQAI
./deploy_whisperx_api.sh
```

The service will be available at `http://192.168.1.76:8181`

## Manual Installation

1. Ensure Docker is installed and running on CQAI
2. Ensure NVIDIA GPU support if using GPU mode (see WhisperX Docker script)
3. **Check existing Python environment** - the deployment script will automatically detect if required packages are already installed:
   ```bash
   python3 -c "import fastapi, uvicorn, pydantic"
   ```
   If this fails, it will install the dependencies automatically.

## Usage

### Starting the Service

```bash
# Start the API service
./start_whisperx_api.sh start

# Check status
./start_whisperx_api.sh status

# View logs
./start_whisperx_api.sh logs

# Stop the service
./start_whisperx_api.sh stop
```

The service runs on `http://192.168.1.76:8181` by default.

### API Endpoints

#### Health Check
```http
GET /health
```

#### Synchronous Transcription
```http
POST /transcribe/sync
Content-Type: multipart/form-data

Parameters:
- file: Audio file (WAV, MP3, M4A, FLAC, OGG)
- language: Language code (default: "en")
- model: Whisper model size (default: "large-v2")
- align_mode: Enable forced alignment (default: false)
- lyrics: Lyrics text for alignment (optional)
```

#### Asynchronous Transcription
```http
POST /transcribe
Content-Type: multipart/form-data

Parameters: Same as sync endpoint

Response:
{
  "status": "accepted",
  "job_id": "job_1",
  "message": "Transcription started. Check status with GET /status/{job_id}"
}
```

#### Check Job Status
```http
GET /status/{job_id}
```

## Integration with Orchestrator

The orchestrator can call the WhisperX API like this:

```python
import requests

# Upload audio file for transcription
files = {'file': open('vocals.wav', 'rb')}
data = {
    'language': 'en',
    'model': 'large-v2',
    'align_mode': 'false'
}

response = requests.post('http://192.168.1.76:8181/transcribe/sync', files=files, data=data)
result = response.json()

# Result contains:
# - transcription: Plain text
# - srt: SRT subtitle format
# - vtt: WebVTT format
# - json_data: Detailed word timestamps
```

## Output Formats

### JSON Data Structure
```json
{
  "transcription": "Hello world this is a test",
  "srt": "1\n00:00:00,000 --> 00:00:04,000\nHello world this is a test\n",
  "vtt": "WEBVTT\n\n1\n00:00:00.000 --> 00:00:04.000\nHello world this is a test\n",
  "json_data": {
    "segments": [
      {
        "start": 0.0,
        "end": 1.0,
        "text": "Hello",
        "words": [
          {"start": 0.0, "end": 0.5, "word": "Hello", "score": 0.95}
        ]
      }
    ]
  }
}
```

## Configuration

- **Host**: 0.0.0.0 (accessible from all interfaces)
- **Port**: 8181
- **Log File**: `whisperx_api.log`
- **PID File**: `whisperx_api.pid`

## Dependencies

- FastAPI
- Uvicorn
- Python 3.8+
- Docker
- WhisperX Docker container

## Troubleshooting

1. **Service won't start**: Check if port 8181 is available
2. **Docker errors**: Ensure Docker daemon is running and user has permissions
3. **GPU issues**: Verify NVIDIA drivers and container toolkit are installed
4. **Large files**: Use asynchronous endpoint for files > 100MB
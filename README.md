# TrackStudio Orchestrator

Go backend API and processing orchestrator for TrackStudio.

## Quick Start

```bash
# Install dependencies
make deps

# Run in development mode (with hot reload)
make dev

# Build and run
make run

# Run tests
make test
```

## Features

- RESTful API for song and queue management
- Background worker for video generation
- Audio analysis integration
- Image generation (LLM + Zimage via CQAI)
- Video composition pipeline
- YouTube upload automation

## Architecture

- **Go API**: Gin framework, SQLite database
- **Queue Worker**: Automatic background processing
- **AI Integration**: CQAI for LLM prompts and image generation
- **Audio Processing**: Local librosa (Python) for audio analysis
- **Video Composition**: FFmpeg for multi-layer video rendering

## Development

### Prerequisites

- Go 1.21+
- Python 3.10+ (with librosa, soundfile, numpy, scipy)
- FFmpeg (with libx264, AAC, filters)
- SQLite 3.31+

### Make Commands

View all available commands:
```bash
make help
```

#### Development Commands
- `make dev` - Run with hot reload (requires air)
- `make run` - Build and run locally
- `make test` - Run all tests
- `make fmt` - Format code
- `make lint` - Run linters

#### Build Commands
- `make build` - Build for current platform
- `make build-linux` - Build for Linux (production)
- `make build-all` - Build for all platforms
- `make clean` - Remove build artifacts

#### Deployment Commands
- `make deploy-mule` - Deploy to mule.nlaakstudios
- `make status-mule` - Check service status
- `make logs-mule` - View service logs
- `make restart-mule` - Restart service

#### Database Commands
- `make db-init` - Initialize database
- `make db-seed` - Seed with test data
- `make db-reset` - Reset database

### Configuration

Environment variables (or `.env` file):
- `ENVIRONMENT` - `development` or `production`
- `CQAI_URL` - CQAI API base URL
- `CQAI_LLM_MODEL` - LLM model name (qwen2.5:7b)
- `CQAI_IMAGE_MODEL` - Image model name (z-image-nsfw)

See `config/config.go` for full configuration options.

## Project Structure

```
.
├── cmd/server/          # Application entry point
├── internal/            # Private application code
│   ├── database/        # Database connection & queries
│   ├── models/          # Data models
│   ├── handlers/        # HTTP handlers
│   ├── services/        # Business logic
│   ├── queue/           # Background job processing
│   └── youtube/         # YouTube API integration
├── pkg/                 # Public/shared packages
│   ├── audio/           # Audio processing (librosa wrapper)
│   ├── video/           # Video composition (FFmpeg)
│   ├── lyrics/          # Lyrics parsing
│   └── image/           # Image generation (CQAI client)
├── config/              # Configuration management
├── data/                # SQLite database files
├── storage/             # Media file storage
│   ├── songs/           # Audio stems
│   ├── videos/          # Generated videos
│   └── temp/            # Temporary files
└── scripts/             # Utility scripts
```

## API Endpoints

See full API documentation in the [docs](../track-studio-docs/api/) folder.

### Core Endpoints
- `GET /api/songs` - List all songs
- `POST /api/songs` - Create new song
- `GET /api/songs/:id` - Get song details
- `PUT /api/songs/:id` - Update song
- `DELETE /api/songs/:id` - Delete song

### Queue Endpoints
- `GET /api/queue` - List queue items
- `POST /api/queue` - Add song to queue
- `GET /api/queue/:id` - Get queue item status
- `DELETE /api/queue/:id` - Cancel queue item

### Processing Endpoints
- `POST /api/process/analyze` - Analyze audio
- `POST /api/process/generate-prompts` - Generate image prompts
- `POST /api/process/generate-images` - Generate images
- `POST /api/process/compose-video` - Compose final video

## Deployment

### Production (mule.nlaakstudios)

```bash
# Build and deploy
make deploy-mule

# Check status
make status-mule

# View logs
make logs-mule
```

The service runs under systemd: `/etc/systemd/system/track-studio-orchestrator.service`

## Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-verbose

# Run specific test
go test -v ./internal/handlers
```

## Troubleshooting

### Hot reload not working
Install air:
```bash
go install github.com/cosmtrek/air@latest
```

### Database locked errors
Stop any running instances:
```bash
pkill -f trackstudio-server
```

### CQAI connection issues
Verify network access:
```bash
curl http://cqai.nlaakstudios/api/health
```

## License

Copyright © 2026 Nlaak Studios. All rights reserved.

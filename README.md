# TrackStudio Orchestrator

Go backend API and processing orchestrator for TrackStudio.

## Features

- RESTful API for song and queue management
- Background worker for video generation
- Audio analysis integration
- Image generation (LLM + Zimage)
- Video composition pipeline
- YouTube upload automation

## Architecture

- **Go API**: Gin framework, SQLite database
- **Queue Worker**: Automatic background processing
- **AI Integration**: CQAI for audio analysis, LLM prompts, image generation

## Getting Started

```bash
# Install dependencies
go mod download

# Run in development mode
go run cmd/server/main.go

# Build for production
go build -o bin/server cmd/server/main.go
```

## Configuration

Environment variables:
- `ENVIRONMENT`: `development` or `production`

See `config/config.go` for full configuration options.

## API Documentation

See [API.md](docs/API.md) for endpoint documentation.

## License

Copyright © 2026 Nlaak Studios. All rights reserved.

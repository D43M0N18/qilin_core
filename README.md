# UGC AI Video Ad Platform

A professional UGC (User Generated Content) ad creation platform that uses AI to analyze product descriptions and images, automatically select appropriate characters, and generate advertisement videos. The platform features a Claude AI-like chat interface with real-time streaming responses, file uploads, conversation management, and video generation capabilities.

## Core Technology Stack

- **Backend:** Go 1.21+, Gin Web Framework, PostgreSQL 15+, Redis 7+
- **Frontend:** React 18+, TypeScript 5+, Vite 5+, TailwindCSS 3+
- **Real-time:** WebSockets (gorilla/websocket)
- **Storage:** AWS S3 / MinIO
- **AI:** Anthropic Claude API (Vision + Text)
- **Authentication:** JWT with refresh tokens
- **Deployment:** Docker + Docker Compose

## Getting Started

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 15+
- Redis 7+
- Node.js 18+ (for frontend)
- Docker and Docker Compose (for deployment)

### Environment Setup

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Update the environment variables in `.env` with your configuration

### Development

1. Install Go dependencies:
   ```bash
   go mod download
   ```

2. Start the development server:
   ```bash
   go run cmd/server/main.go
   ```

### API Documentation

API documentation can be found in the `/docs` directory.

## Project Structure

```
.
├── cmd/
│   └── server/              # Application entry point
├── internal/
│   ├── api/                 # HTTP/WebSocket handlers
│   ├── models/              # Data models
│   ├── services/            # Business logic
│   ├── database/            # Database layer
│   ├── config/             # Configuration
│   └── utils/              # Utilities
└── pkg/                    # Public packages
```

## License

Copyright © 2025 Qilin. All rights reserved.

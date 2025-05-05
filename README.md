# Gitlab Orchestrator Telegram Bot

A Telegram bot for orchestrating and managing GitLab environments, pipelines, and stands.

## Project Structure

The project consists of two main components:

1. **Telegram Bot (gitlab-orchestrator-bot)**: A Go application that provides a Telegram interface for users
2. **Backend API (gitlab-orchestrator-back)**: A Go server that handles GitLab integration and database operations

## Configuration

### Environment Variables

The application uses environment variables for configuration. These can be set directly or provided via a `.env` file.

#### Required Variables

- `TOKEN`: Telegram Bot API token from BotFather
- `BACKEND_URL`: URL of the backend API (e.g., http://localhost:8080/api/v1)

#### Optional Variables

- `LOG_LEVEL`: Logging level (debug, info, warn, error) - default: info
- `ENV`: Environment type (development, staging, production) - default: development
- `DOMAIN`: Custom domain suffix for stands - default: .example.com

### Using .env File

Create a `.env` file in the project root based on the example:

```bash
# Copy example file
cp .env.example .env

# Edit the .env file with your settings
nano .env
```

## Development

### Prerequisites

- Go 1.21 or higher
- PostgreSQL database (for backend)

### Running the Bot

```bash
cd gitlab-orchestrator-tg-bot
go run main.go
```

### Running the Backend

```bash
cd gitlab-orchestrator-back
go run cmd/api/main.go
```

## Docker Deployment

The project includes a Dockerfile for containerized deployment:

```bash
docker build -t gitlab-orchestrator-bot .
docker run -e TOKEN=your_token -e BACKEND_URL=your_backend_url gitlab-orchestrator-bot
```

For detailed documentation of each component, refer to the README files in the respective directories.

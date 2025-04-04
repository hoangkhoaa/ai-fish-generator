#!/bin/bash

# Exit on any error
set -e

echo "Starting Fish Generate service deployment to staging..."

# Ensure .env file exists
if [ ! -f .env ]; then
  if [ -f .env.example ]; then
    echo "Warning: .env file not found but .env.example exists."
    echo "Creating .env file from .env.example"
    cp .env.example .env
    echo "Please edit .env with your actual API keys before continuing."
    echo "Press Enter to continue after editing, or Ctrl+C to cancel..."
    read
  else
    echo "Error: Neither .env nor .env.example found!"
    echo "Please create a .env file with your API keys before deploying."
    exit 1
  fi
fi

# Verify required environment variables in .env
echo "Checking required environment variables..."
required_vars=("GEMINI_API_KEY" "OPENWEATHER_API_KEY" "NEWSAPI_KEY" "METALPRICE_API_KEY")
missing_vars=false

for var in "${required_vars[@]}"; do
  if ! grep -q "^${var}=" .env || grep -q "^${var}=$" .env; then
    echo "Missing or invalid ${var} in .env file"
    missing_vars=true
  fi
done

if [ "$missing_vars" = true ]; then
  echo "Please update your .env file with valid API keys."
  exit 1
fi

# Determine docker compose command
if docker compose version &> /dev/null; then
  DOCKER_COMPOSE="docker compose"
else
  if ! command -v docker-compose &> /dev/null; then
    echo "Error: Neither 'docker compose' nor 'docker-compose' is available."
    exit 1
  fi
  DOCKER_COMPOSE="docker-compose"
fi

echo "Using $DOCKER_COMPOSE for deployment"

# Pull latest code (if in a git repository)
if [ -d .git ]; then
  echo "Pulling latest code..."
  git pull
fi

# Stop any running containers first
echo "Stopping any running containers..."
$DOCKER_COMPOSE down

# Build and start containers
echo "Building and starting Docker containers..."
$DOCKER_COMPOSE build --no-cache
$DOCKER_COMPOSE up -d

# Wait for MongoDB to be ready
echo "Waiting for MongoDB to initialize..."
sleep 10

# Check if containers are running
echo "Checking service status..."
if $DOCKER_COMPOSE ps | grep -q "fish-generate" && $DOCKER_COMPOSE ps | grep -q "fish-mongodb"; then
  echo "Fish Generate service is running!"
  echo "You can check logs with: $DOCKER_COMPOSE logs -f app"
else
  echo "Error: Service failed to start. Check logs with: $DOCKER_COMPOSE logs"
  $DOCKER_COMPOSE logs app
  exit 1
fi

echo "Deployment completed successfully!" 
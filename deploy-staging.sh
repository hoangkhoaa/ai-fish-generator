#!/bin/bash

# Exit on any error
set -e

echo "Starting Fish Generate service deployment to staging..."

# Ensure .env file exists
if [ ! -f .env ]; then
  echo "Error: .env file not found!"
  echo "Please create a .env file based on .env.example with your API keys."
  exit 1
fi

# Ensure docker is installed
if ! command -v docker &> /dev/null || ! command -v docker-compose &> /dev/null; then
  echo "Error: Docker and Docker Compose are required but not installed."
  exit 1
fi

# Pull latest code (if in a git repository)
if [ -d .git ]; then
  echo "Pulling latest code..."
  git pull
fi

# Build and start containers
echo "Building and starting Docker containers..."
docker-compose down
docker-compose build --no-cache
docker-compose up -d

# Check if containers are running
echo "Checking service status..."
if docker-compose ps | grep -q "fish-generate"; then
  echo "Fish Generate service is running!"
  echo "You can check logs with: docker-compose logs -f app"
else
  echo "Error: Service failed to start. Check logs with: docker-compose logs app"
  exit 1
fi

echo "Deployment completed successfully!" 
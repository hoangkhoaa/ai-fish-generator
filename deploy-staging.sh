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

# Stop any running containers and remove volumes if needed
echo "Stopping any running containers..."
$DOCKER_COMPOSE down

# Ask if volumes should be removed (useful for auth issues)
read -p "Do you want to remove volumes to start fresh? (useful for MongoDB auth issues) [y/N]: " remove_volumes
if [[ "$remove_volumes" =~ ^([yY][eE][sS]|[yY])$ ]]; then
  echo "Removing volumes for a fresh start..."
  $DOCKER_COMPOSE down -v
fi

# Build and start containers
echo "Building and starting Docker containers..."
$DOCKER_COMPOSE build --no-cache
$DOCKER_COMPOSE up -d

# Wait for MongoDB to be ready
echo "Waiting for MongoDB to initialize (this may take up to 30 seconds)..."
sleep 20

# Check if containers are running
echo "Checking service status..."
if $DOCKER_COMPOSE ps | grep -q "fish-generate" && $DOCKER_COMPOSE ps | grep -q "fish-mongodb"; then
  echo "Containers are running. Checking logs for MongoDB connection..."
  
  # Check logs for successful MongoDB connection
  if $DOCKER_COMPOSE logs app | grep -q "MongoDB connection failed"; then
    echo "⚠️ Warning: MongoDB connection issues detected. See logs for details."
    echo "You may need to:"
    echo "  1. Update your MONGO_URI in docker-compose.yml"
    echo "  2. Run this script again with the option to remove volumes"
    $DOCKER_COMPOSE logs app | grep -A 5 -B 5 "MongoDB connection"
  else
    echo "✅ Services appear to be running correctly."
  fi
  
  echo "Fish Generate service deployment completed."
  echo "You can check detailed logs with: $DOCKER_COMPOSE logs -f app"
else
  echo "❌ Error: Service failed to start. Checking logs..."
  $DOCKER_COMPOSE logs app
  exit 1
fi 
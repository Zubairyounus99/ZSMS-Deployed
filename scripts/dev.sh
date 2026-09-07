#!/usr/bin/env bash
set -e

echo ">>> Starting ZSMS Local Development Environment..."

if [ ! -f ".env" ]; then
    echo ">>> .env not found, copying from .env.example..."
    cp .env.example .env
fi

if command -v docker &> /dev/null; then
    echo ">>> Launching Docker Compose stack (dev)..."
    docker compose -f docker-compose.dev.yml up -d
else
    echo ">>> Docker not found. Please install Docker or start Docker Engine."
    exit 1
fi

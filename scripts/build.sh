#!/usr/bin/env bash
set -e

echo ">>> Building ZSMS Monorepo Artifacts..."

# 1. Frontend Build Check
if command -v npm &> /dev/null; then
    echo ">>> Building Nuxt Web Frontend..."
    cd web && npm run build && cd ..
fi

# 2. Go Backend Build Check
if command -v go &> /dev/null; then
    echo ">>> Building Go Backend..."
    cd backend && go build ./... && cd ..
fi

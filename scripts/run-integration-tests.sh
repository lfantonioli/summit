#!/bin/bash
set -e

echo "Building Alpine integration test image..."
docker build -f Dockerfile.integration -t summit-integration .

echo "Running integration tests..."
docker run --rm summit-integration

echo "Tests completed successfully!"
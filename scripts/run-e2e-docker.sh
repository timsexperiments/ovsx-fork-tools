#!/bin/bash
set -e

# Ensure we are running from the project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"


echo "Building Docker image..."
docker build -t ovsx-e2e -f tests/e2e/Dockerfile .

# Get the current working directory (which is now project root)
WORK_DIR=$(pwd)

# Run the container
# -v /var/run/docker.sock:/var/run/docker.sock: Mount docker socket for act
# -v "$WORK_DIR":"$WORK_DIR": Mount project to same path as host
# -w "$WORK_DIR": Set working directory to the mounted path
# -e E2E=true: Enable E2E tests
# We build the binary INSIDE the container to ensure it matches the container's architecture (Linux/AMD64)
# and to avoid relying on the host's Go environment.
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$WORK_DIR":"$WORK_DIR" \
  -w "$WORK_DIR" \
  -e E2E=true \
  ovsx-e2e \
  /bin/bash -c "go build -o ovsx-setup main.go && go test -v ./tests/e2e/..."

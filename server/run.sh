#!/bin/bash

# NanoLog Server Build & Run Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== NanoLog Server ===${NC}"

# Parse arguments
ACTION="${1:-run}"
ARGS="${@:2}"

# If first arg is a flag, default to 'run' and use all args
if [[ "$ACTION" == -* ]]; then
    ARGS="$@"
    ACTION="run"
fi

case "$ACTION" in
    build)
        echo -e "${YELLOW}Building...${NC}"
        go build -o bin/nanolog cmd/nanolog/main.go
        echo -e "${GREEN}Build complete: bin/nanolog${NC}"
        ;;
    run)
        echo -e "${YELLOW}Running in development mode...${NC}"
        go run cmd/nanolog/main.go $ARGS
        ;;
    start)
        echo -e "${YELLOW}Building and starting...${NC}"
        go build -o bin/nanolog cmd/nanolog/main.go
        ./bin/nanolog $ARGS
        ;;
    test)
        echo -e "${YELLOW}Running tests...${NC}"
        go test ./... -v
        ;;
    tidy)
        echo -e "${YELLOW}Tidying modules...${NC}"
        go mod tidy
        echo -e "${GREEN}Done${NC}"
        ;;
    *)
        echo "Usage: $0 {build|run|start|test|tidy}"
        echo ""
        echo "Commands:"
        echo "  build  - Compile to bin/nanolog"
        echo "  run    - Run in development mode (default)"
        echo "  start  - Build and run"
        echo "  test   - Run all tests"
        echo "  tidy   - Run go mod tidy"
        exit 1
        ;;
esac

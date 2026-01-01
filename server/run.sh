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
        CGO_ENABLED=0 go build -o bin/nanolog cmd/nanolog/main.go
        echo -e "${GREEN}Build complete: bin/nanolog${NC}"
        ;;
    run)
        echo -e "${YELLOW}Running in development mode...${NC}"
        CGO_ENABLED=0 go run cmd/nanolog/main.go $ARGS
        ;;
    start)
        echo -e "${YELLOW}Building and starting...${NC}"
        CGO_ENABLED=0 go build -o bin/nanolog cmd/nanolog/main.go
        ./bin/nanolog $ARGS
        ;;
    standalone)
        echo -e "${YELLOW}Starting Standalone Mode (Full Functionality)...${NC}"
        CGO_ENABLED=0 go build -o bin/nanolog cmd/nanolog/main.go
        ./bin/nanolog --role=standalone $ARGS
        ;;
    console)
        echo -e "${YELLOW}Starting Console Mode (Management & Query Aggregation)...${NC}"
        if [[ -z "$DATA_NODES" ]]; then
            echo -e "${YELLOW}Tip: Set DATA_NODES env or pass --data-nodes to specify ingester nodes.${NC}"
        fi
        CGO_ENABLED=0 go build -o bin/nanolog cmd/nanolog/main.go
        ./bin/nanolog --role=console $ARGS
        ;;
    ingester)
        echo -e "${YELLOW}Starting Ingester Mode (Storage & Local Query)...${NC}"
        CGO_ENABLED=0 go build -o bin/nanolog cmd/nanolog/main.go
        ./bin/nanolog --role=ingester $ARGS
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
        echo "Usage: $0 {build|run|start|standalone|console|ingester|test|tidy} [options]"
        echo ""
        echo "Commands:"
        echo "  build      - Compile to bin/nanolog"
        echo "  run        - Run in development mode (default)"
        echo "  start      - Build and run with custom args"
        echo "  standalone - Start in full standalone mode"
        echo "  console    - Start as Console node (needs --data-nodes)"
        echo "  ingester   - Start as Ingester node"
        echo "  test       - Run all tests"
        echo "  tidy       - Run go mod tidy"
        echo ""
        echo "Examples:"
        echo "  $0 standalone --port 8080"
        echo "  $0 console --data-nodes=http://node1:8081,http://node2:8081"
        echo "  $0 ingester --port 8081"
        exit 1
        ;;
esac

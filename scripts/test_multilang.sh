#!/bin/bash

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}=== NanoLog Multi-Language Verification ===${NC}"

# Check Server
echo -e "\n${YELLOW}Checking NanoLog Server...${NC}"
if ! lsof -i :8088 > /dev/null; then
    echo -e "${RED}Error: NanoLog server is NOT running on port 8088.${NC}"
    echo "Please start the server first using: ./server/run.sh standalone"
    exit 1
else
     echo -e "${GREEN}Server is OK.${NC}"
fi

# 1. Python SDK Verification
echo -e "\n${YELLOW}1. Running Python SDK Example...${NC}"
if command -v python3 &> /dev/null; then
    export PYTHONPATH=$PYTHONPATH:$(pwd)/sdks/python
    if python3 sdks/python/nanolog/example.py; then
        echo -e "${GREEN}Python SDK: Success${NC}"
    else
        echo -e "${RED}Python SDK: Failed${NC}"
    fi
else
     echo -e "${RED}Python3 not found. Skipping...${NC}"
fi

# 2. Go SDK Verification
echo -e "\n${YELLOW}2. Running Go SDK Example...${NC}"
if command -v go &> /dev/null; then
    cd sdks/go/nanolog/example
    # Ensure deps are ready
    if [ ! -f "go.mod" ]; then
        go mod init example > /dev/null 2>&1
        go mod edit -replace github.com/coffersTech/nanolog/sdks/go/nanolog=../ > /dev/null 2>&1
    fi
    go mod tidy > /dev/null 2>&1
    
    if go run main.go; then
        echo -e "${GREEN}Go SDK: Success${NC}"
    else
        echo -e "${RED}Go SDK: Failed${NC}"
    fi
    cd - > /dev/null
else
    echo -e "${RED}Go command not found. Skipping...${NC}"
fi

# 3. Final Prompt
echo -e "\n${GREEN}=== Verification Complete ===${NC}"
echo -e "Please visit the Web Dashboard to verify instances:"
echo -e "URL: ${YELLOW}http://localhost:8088/instances${NC}"
echo -e "You should see:"
echo -e " - ${GREEN}python-script${NC}"
echo -e " - ${GREEN}go-service${NC}"
echo -e " - ${GREEN}my-order-service${NC} (if Java was run previously)"

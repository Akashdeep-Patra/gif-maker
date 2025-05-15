#!/bin/bash

# Colors for terminal output
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Use test.gif as the invalid input file
INVALID_FILE="test.gif"
BINARY_PATH="./build/gif-maker"
OUTPUT_FILE="output_should_fail.gif"

# Check if the binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo -e "${RED}Error: Binary not found at $BINARY_PATH${NC}"
    echo -e "${YELLOW}Please build the project first${NC}"
    exit 1
fi

# Create a test.gif file if it doesn't exist
if [ ! -f "$INVALID_FILE" ]; then
    echo -e "${YELLOW}Creating dummy $INVALID_FILE file for testing...${NC}"
    # Create a small empty gif file
    echo -e "GIF89a\x01\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x21\xf9\x04\x01\x00\x00\x00\x00\x2c\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x01\x44\x00\x3b" > "$INVALID_FILE"
fi

# Run the binary with the invalid file
echo -e "${BLUE}Testing with invalid file format: $INVALID_FILE${NC}"
echo -e "${BLUE}Attempting to convert a GIF file (should fail)...${NC}"
$BINARY_PATH convert --input "$INVALID_FILE" --output "$OUTPUT_FILE" --fps 10 --quality 80 --no-progress

# The command should fail, so we invert the exit code check
if [ $? -ne 0 ]; then
    echo -e "${GREEN}\033[1m✓ Test passed! Application correctly rejected the invalid file format.${NC}"
    exit 0
else
    echo -e "${RED}✗ Test failed! Application accepted a non-video file format.${NC}"
    # Clean up the output file if it was created
    if [ -f "$OUTPUT_FILE" ]; then
        echo -e "${YELLOW}Removing unexpected output file...${NC}"
        rm -f "$OUTPUT_FILE"
    fi
    exit 1
fi 
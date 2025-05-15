#!/bin/bash

# Colors for terminal output
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if video filename was provided as an argument
if [ "$#" -ne 1 ]; then
    echo -e "${RED}Error: Missing input video file argument${NC}"
    echo -e "${YELLOW}Usage: $0 path/to/video.mp4${NC}"
    exit 1
fi

VIDEO_FILE=$1
BINARY_PATH="./build/gif-maker"
OUTPUT_FILE="${VIDEO_FILE%.*}.gif"

# Check if the binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo -e "${RED}Error: Binary not found at $BINARY_PATH${NC}"
    echo -e "${YELLOW}Please build the project first${NC}"
    exit 1
fi

# Check if the video file exists
if [ ! -f "$VIDEO_FILE" ]; then
    echo -e "${RED}Error: Test video file $VIDEO_FILE not found${NC}"
    echo -e "${YELLOW}Please provide a valid video file${NC}"
    exit 1
fi

# Run the binary with the video file
echo -e "${BLUE}Running gif-maker with $VIDEO_FILE...${NC}"
$BINARY_PATH convert --input "$VIDEO_FILE" --output "$OUTPUT_FILE" --fps 10 --quality 80 --no-progress

# Check if execution was successful
if [ $? -eq 0 ]; then
    echo -e "${GREEN}\033[1m✓ Test successful! Application ran without errors.${NC}"
    # Check if GIF file was created
    if [ -f "$OUTPUT_FILE" ]; then
        echo -e "${GREEN}✓ Output GIF file created: $OUTPUT_FILE${NC}"
        
        # Cleanup the output file
        echo -e "${BLUE}Cleaning up test artifacts...${NC}"
        rm -f "$OUTPUT_FILE"
        echo -e "${GREEN}✓ Removed output file${NC}"
    else
        echo -e "${YELLOW}! No output GIF file found with name: $OUTPUT_FILE${NC}"
    fi
    exit 0
else
    echo -e "${RED}✗ Test failed! Application exited with an error.${NC}"
    exit 1
fi 
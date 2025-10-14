#!/bin/bash

# A script to test various features of the MontageGo application.
#
# Before running:
# 1. Make sure you have compiled the application (e.g., using scripts/build.sh).
# 2. Place a video file named "中文BigBuckBunny.mp4" in the `tests/videos` directory.
# 3. Make this script executable: chmod +x scripts/run_tests.sh
#
# This script will generate several test_*.jpg files in a new `tests/outputs` directory.

set -e # Exit immediately if a command exits with a non-zero status.
set -x # Print commands and their arguments as they are executed.

# --- Configuration ---
# Path to the compiled binary
BINARY="./builds/MontageGo-darwin-arm64"
# Path to the test video file
VIDEO_FILE="tests/videos/中文BigBuckBunny.mp4"
# Path to a common font file on macOS for text rendering tests
FONT_FILE="/System/Library/Fonts/STHeiti Light.ttc"
# Output directory for generated images
OUTPUT_DIR="tests/outputs"

# --- Setup ---
# Check if binary exists
if [ ! -f "$BINARY" ]; then
    echo "Error: Binary not found at $BINARY. Please build the project first."
    exit 1
fi

# Check if video file exists
if [ ! -f "$VIDEO_FILE" ]; then
    echo "Error: Test video not found at '$VIDEO_FILE'."
    echo "Please place a video with this name in the project root directory."
    exit 1
fi

# Check if font file exists for text-related tests
if [ ! -f "$FONT_FILE" ]; then
    echo "Warning: Default macOS font not found at '$FONT_FILE'. Skipping text-related tests."
    # We will proceed without text tests if the font is not there.
    FONT_FILE=""
fi

# Create a fresh output directory
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"
echo "--- Starting MontageGo Tests ---"

# --- Test Cases ---

echo "\n[1/7] Testing default settings..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_1_default.jpg" --font-file "$FONT_FILE"

echo "\n[2/7] Testing different grid layout (2x3) with large padding and margin..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_2_layout.jpg" --font-file "$FONT_FILE" \
    -c 2 -r 3 --padding 20 --margin 50

echo "\n[3/7] Testing custom colors (lime text, magenta shadow, navy background)..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_3_colors.jpg" --font-file "$FONT_FILE" \
    --font-color "lime" --shadow-color "#FF00FF" --bg-color "navy"

echo "\n[4/7] Testing high JPEG quality..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_4_high_quality.jpg" --font-file "$FONT_FILE" \
    --jpeg-quality 1

echo "\n[5/7] Testing low JPEG quality..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_5_low_quality.jpg" --font-file "$FONT_FILE" \
    --jpeg-quality 31

echo "\n[6/7] Testing with no text rendering (empty font-file)..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_6_no_text.jpg" --header 0 --font-file ""

echo "\n[7/7] Testing output to stdout (piping to a file)..."
"$BINARY" "$VIDEO_FILE" -o - --font-file "$FONT_FILE" > "$OUTPUT_DIR/test_7_piped.jpg"


# --- Completion ---
set +x
echo "\n--- All tests completed successfully! ---"
echo "Check the '$OUTPUT_DIR' directory for the generated montage images."

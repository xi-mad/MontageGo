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

echo "\n[1/13] Testing default settings..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_01_default.jpg" --font-file "$FONT_FILE"

echo "\n[2/13] Testing different grid layout (2x3) with large padding and margin..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_02_layout_2x3.jpg" --font-file "$FONT_FILE" \
    -c 2 -r 3 --padding 20 --margin 50

echo "\n[3/13] Testing large grid layout (6x6)..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_03_layout_6x6.jpg" --font-file "$FONT_FILE" \
    -c 6 -r 6 --padding 10 --margin 30

echo "\n[4/13] Testing single column layout (1x8)..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_04_layout_1x8.jpg" --font-file "$FONT_FILE" \
    -c 1 -r 8 --padding 10 --margin 15

echo "\n[5/13] Testing custom colors (lime text, magenta shadow, navy background)..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_05_colors.jpg" --font-file "$FONT_FILE" \
    --font-color "lime" --shadow-color "#FF00FF" --bg-color "navy"

echo "\n[6/13] Testing high JPEG quality..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_06_high_quality.jpg" --font-file "$FONT_FILE" \
    --jpeg-quality 1

echo "\n[7/13] Testing low JPEG quality..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_07_low_quality.jpg" --font-file "$FONT_FILE" \
    --jpeg-quality 31

echo "\n[8/13] Testing with no text rendering (empty font-file, no header)..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_08_no_text.jpg" --header 0 --font-file ""

echo "\n[9/13] Testing small thumbnail size (320px width)..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_09_small_thumbs.jpg" --font-file "$FONT_FILE" \
    --thumb-width 320

echo "\n[10/13] Testing minimal padding and margin..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_10_minimal_spacing.jpg" --font-file "$FONT_FILE" \
    --padding 0 --margin 5

echo "\n[11/13] Testing custom header height..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_11_custom_header.jpg" --font-file "$FONT_FILE" \
    --header 200

echo "\n[12/13] Testing quiet mode (should produce no logs)..."
"$BINARY" "$VIDEO_FILE" -o "$OUTPUT_DIR/test_12_quiet.jpg" --font-file "$FONT_FILE" --quiet

echo "\n[13/13] Testing output to stdout (piping to a file)..."
"$BINARY" "$VIDEO_FILE" -o - --font-file "$FONT_FILE" > "$OUTPUT_DIR/test_13_piped.jpg"


# --- Completion ---
set +x
echo "\n--- All tests completed successfully! ---"
echo "Check the '$OUTPUT_DIR' directory for the generated montage images."

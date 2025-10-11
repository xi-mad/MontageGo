#!/bin/bash
#
# This script compiles the MontageGo application for Windows, Linux, and macOS.
#

echo "🚀 Starting cross-platform build process for MontageGo..."

# Define the output directory for the binaries
OUTPUT_DIR="builds"

# Create the output directory if it doesn't exist, and clear its contents
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Define the target platforms in a "OS/ARCH" format
PLATFORMS="windows/amd64 linux/amd64 darwin/amd64 darwin/arm64"

# Get the main package path from the current directory
PACKAGE_PATH="./cmd/montagego"

for PLATFORM in $PLATFORMS
do
    # Split the platform string into OS and Architecture
    GOOS=${PLATFORM%/*}
    GOARCH=${PLATFORM#*/}
    
    # Set the output binary name
    BINARY_NAME="MontageGo"
    if [ "$GOOS" = "windows" ]; then
        BINARY_NAME+=".exe"
    fi
    
    OUTPUT_NAME="$OUTPUT_DIR/${BINARY_NAME}-${GOOS}-${GOARCH}"

    echo "Building for $GOOS/$GOARCH..."
    
    # Set the environment variables for cross-compilation and run the build command
    env GOOS="$GOOS" GOARCH="$GOARCH" go build -o "$OUTPUT_NAME" "$PACKAGE_PATH"
    
    # Check if the build command was successful
    if [ $? -ne 0 ]; then
        echo "❌ Build failed for $GOOS/$GOARCH"
        # Optional: exit on first failure
        # exit 1 
    else
        echo "✅ Successfully built $OUTPUT_NAME"
    fi
done

echo ""
echo "🎉 All builds finished!"
echo "Binaries are located in the '$OUTPUT_DIR' directory."

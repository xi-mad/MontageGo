#!/bin/bash
#
# This script automates the release process for MontageGo.
# It will:
# 1. Validate the version format
# 2. Create a git tag
# 3. Build binaries for all platforms
# 4. Guide you to create a GitHub release with the build artifacts
#
# Usage: ./scripts/release.sh <version>
# Example: ./scripts/release.sh v1.0.0
#

set -e  # Exit on error

# --- Configuration ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BUILD_SCRIPT="$SCRIPT_DIR/build.sh"
BUILDS_DIR="$PROJECT_ROOT/builds"

# --- Functions ---
show_usage() {
    echo "Usage: $0 <version>"
    echo ""
    echo "Examples:"
    echo "  $0 v1.0.0"
    echo "  $0 v1.2.3"
    echo ""
    echo "The version must start with 'v' followed by semantic versioning (e.g., v1.0.0)"
}

validate_version() {
    local version=$1
    if [[ ! $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "❌ Error: Invalid version format '$version'"
        echo "Version must follow semantic versioning: v<major>.<minor>.<patch>"
        echo "Example: v1.0.0"
        return 1
    fi
    return 0
}

check_git_status() {
    if [[ -n $(git status -s) ]]; then
        echo "⚠️  Warning: You have uncommitted changes:"
        git status -s
        echo ""
        read -p "Do you want to continue anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Aborted."
            exit 1
        fi
    fi
}

check_tag_exists() {
    local version=$1
    if git rev-parse "$version" >/dev/null 2>&1; then
        echo "❌ Error: Tag '$version' already exists!"
        echo "Existing tags:"
        git tag -l
        exit 1
    fi
}

create_tag() {
    local version=$1
    echo "📝 Creating git tag '$version'..."
    read -p "Enter release description (optional): " description
    
    if [[ -z "$description" ]]; then
        git tag "$version"
    else
        git tag -a "$version" -m "$description"
    fi
    
    echo "✅ Tag '$version' created successfully"
}

build_binaries() {
    echo ""
    echo "🔨 Building binaries..."
    if [[ ! -x "$BUILD_SCRIPT" ]]; then
        echo "❌ Error: Build script not found or not executable: $BUILD_SCRIPT"
        exit 1
    fi
    
    "$BUILD_SCRIPT"
    
    if [[ $? -ne 0 ]]; then
        echo "❌ Build failed!"
        exit 1
    fi
}

show_release_info() {
    local version=$1
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🎉 Release preparation completed!"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "Version: $version"
    echo "Builds location: $BUILDS_DIR/"
    echo ""
    echo "Generated artifacts (upload these to the GitHub Release):"
    ls -lh "$BUILDS_DIR"
    echo ""
    echo "Includes:"
    echo "  - Per-platform binaries (MontageGo-<os>-<arch>[.exe])"
    echo "  - config.sample.yaml"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Next steps:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "1. Push the tag to GitHub:"
    echo "   git push origin $version"
    echo ""
    echo "2. Create a GitHub Release:"
    echo "   - Go to: https://github.com/xi-mad/MontageGo/releases/new"
    echo "   - Select tag: $version"
    echo "   - Upload files from: $BUILDS_DIR/"
    echo "   - Publish the release"
    echo ""
    echo "Or push the tag and GitHub will allow you to create a release from it."
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# --- Main Script ---
main() {
    cd "$PROJECT_ROOT"
    
    # Check arguments
    if [[ $# -ne 1 ]]; then
        echo "❌ Error: Version argument required"
        echo ""
        show_usage
        exit 1
    fi
    
    VERSION=$1
    
    echo "🚀 Starting release process for MontageGo"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    
    # Validate version format
    validate_version "$VERSION"
    
    # Check git status
    check_git_status
    
    # Check if tag already exists
    check_tag_exists "$VERSION"
    
    # Create the tag
    create_tag "$VERSION"
    
    # Build binaries
    build_binaries
    
    # Show release information
    show_release_info "$VERSION"
}

# Run main function
main "$@"


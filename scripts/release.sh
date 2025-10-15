#!/bin/bash
#
# Minimal release helper: create and push a git tag.
# Usage:
#   ./scripts/release.sh vX.Y.Z [message]
# Example:
#   ./scripts/release.sh v0.0.3 "Initial public release"
#
set -e

show_usage() {
    echo "Usage: $0 vX.Y.Z [message]"
}

validate_version() {
    local version=$1
    [[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
        echo "❌ Invalid version: '$version' (expected v<major>.<minor>.<patch>)";
        exit 1;
    }
}

check_tag_exists() {
    local version=$1
    if git rev-parse "$version" >/dev/null 2>&1; then
        echo "❌ Tag '$version' already exists."
        exit 1
    fi
}

main() {
    if [[ $# -lt 1 ]]; then
        show_usage
        exit 1
    fi

    VERSION=$1
    MESSAGE=${2:-}

    validate_version "$VERSION"
    check_tag_exists "$VERSION"

    if [[ -n $(git status -s) ]]; then
        echo "⚠️  You have uncommitted changes (these won't be part of the tag)."
    fi

    echo "📝 Creating tag $VERSION ..."
    if [[ -n "$MESSAGE" ]]; then
        git tag -a "$VERSION" -m "$MESSAGE"
    else
        git tag "$VERSION"
    fi

    echo "🚀 Pushing tag $VERSION to origin ..."
    git push origin "$VERSION"

    echo "✅ Done. GitHub Actions will build and publish artifacts for '$VERSION' automatically."
}

main "$@"


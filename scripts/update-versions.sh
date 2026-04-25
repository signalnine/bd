#!/bin/bash
set -e

# =============================================================================
# Quick version bump utility (no git operations)
# =============================================================================
#
# Updates version numbers across all bd components without any git
# operations. Use this for local testing or when you want manual control
# over commits.
#
# For the full release flow, see docs/RELEASING.md.
#
# =============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

if [ $# -ne 1 ]; then
    echo "Usage: $0 <version>"
    echo ""
    echo "Updates version numbers across all components (no git operations)."
    echo ""
    echo "Example: $0 0.47.1"
    echo ""
    echo "For the full release flow, see docs/RELEASING.md."
    exit 1
fi

NEW_VERSION=$1

# Validate semantic versioning
if ! [[ $NEW_VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}Error: Invalid version format '$NEW_VERSION'${NC}"
    echo "Expected: MAJOR.MINOR.PATCH (e.g., 0.47.1)"
    exit 1
fi

# Check we're in repo root
if [ ! -f "cmd/bd/version.go" ]; then
    echo -e "${RED}Error: Must run from repository root${NC}"
    exit 1
fi

# Get current version
CURRENT_VERSION=$(grep 'Version = ' cmd/bd/version.go | sed 's/.*"\(.*\)".*/\1/')
echo -e "${YELLOW}Bumping: $CURRENT_VERSION → $NEW_VERSION${NC}"
echo ""

# Cross-platform sed helper
update_file() {
    local file=$1
    local old=$2
    local new=$3
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s|$old|$new|g" "$file"
    else
        sed -i "s|$old|$new|g" "$file"
    fi
}

echo "Updating version files..."

# 1. cmd/bd/version.go
echo "  • cmd/bd/version.go"
update_file "cmd/bd/version.go" "Version = \"$CURRENT_VERSION\"" "Version = \"$NEW_VERSION\""

# 2. Plugin JSON files
echo "  • .claude-plugin/*.json"
update_file "claude-plugin/.claude-plugin/plugin.json" "\"version\": \"$CURRENT_VERSION\"" "\"version\": \"$NEW_VERSION\""
update_file ".claude-plugin/marketplace.json" "\"version\": \"$CURRENT_VERSION\"" "\"version\": \"$NEW_VERSION\""

# 5. README badge
echo "  • README.md"
update_file "README.md" "Alpha (v$CURRENT_VERSION)" "Alpha (v$NEW_VERSION)"

# 6. default.nix
echo "  • default.nix"
update_file "default.nix" "version = \"$CURRENT_VERSION\";" "version = \"$NEW_VERSION\";"

# 7. Hook templates — now generated dynamically by cmd/bd/hooks.go using the
# Version constant from version.go. No template files to update.
# (Previously updated cmd/bd/templates/hooks/* which no longer exist.)

# 8. Windows PE resource metadata
echo "  • cmd/bd/winres/winres.json"
update_file "cmd/bd/winres/winres.json" "\"file_version\": \"$CURRENT_VERSION\"" "\"file_version\": \"$NEW_VERSION\""
update_file "cmd/bd/winres/winres.json" "\"product_version\": \"$CURRENT_VERSION\"" "\"product_version\": \"$NEW_VERSION\""
update_file "cmd/bd/winres/winres.json" "\"FileVersion\": \"$CURRENT_VERSION\"" "\"FileVersion\": \"$NEW_VERSION\""
update_file "cmd/bd/winres/winres.json" "\"ProductVersion\": \"$CURRENT_VERSION\"" "\"ProductVersion\": \"$NEW_VERSION\""
echo "  • cmd/bd/winres/manifest.xml"
update_file "cmd/bd/winres/manifest.xml" "version=\"$CURRENT_VERSION.0\"" "version=\"$NEW_VERSION.0\""

echo ""
echo -e "${GREEN}✓ Versions updated to $NEW_VERSION${NC}"
echo ""
echo "Changed files:"
git diff --stat 2>/dev/null || true
echo ""
echo "Next steps:"
echo "  • Update CHANGELOG.md with release notes"
echo "  • git tag v$NEW_VERSION && git push origin v$NEW_VERSION (see docs/RELEASING.md)"

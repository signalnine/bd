#!/bin/bash
# Check that all version files are in sync
# Run this before committing version bumps

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# Get the canonical version from version.go
CANONICAL=$(grep 'Version = ' cmd/bd/version.go | sed 's/.*"\(.*\)".*/\1/')

if [ -z "$CANONICAL" ]; then
    echo -e "${RED}❌ Could not read version from cmd/bd/version.go${NC}"
    exit 1
fi

echo "Canonical version (from version.go): $CANONICAL"
echo ""

MISMATCH=0

check_version() {
    local file=$1
    local version=$2
    local description=$3

    if [ "$version" != "$CANONICAL" ]; then
        echo -e "${RED}❌ $description: $version (expected $CANONICAL)${NC}"
        MISMATCH=1
    else
        echo -e "${GREEN}✓ $description: $version${NC}"
    fi
}

# Check all version files
check_version "claude-plugin/.claude-plugin/plugin.json" \
    "$(jq -r '.version' claude-plugin/.claude-plugin/plugin.json 2>/dev/null)" \
    "Claude plugin.json"

check_version ".claude-plugin/marketplace.json" \
    "$(jq -r '.plugins[0].version' .claude-plugin/marketplace.json 2>/dev/null)" \
    "Claude marketplace.json"

echo ""

if [ $MISMATCH -eq 1 ]; then
    echo -e "${RED}❌ Version mismatch detected!${NC}"
    echo ""
    echo "Run: scripts/update-versions.sh $CANONICAL"
    echo "Or manually update the mismatched files."
    exit 1
else
    echo -e "${GREEN}✓ All versions match: $CANONICAL${NC}"
fi

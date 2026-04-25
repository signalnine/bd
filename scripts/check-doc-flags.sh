#!/bin/bash
# check-doc-flags.sh — Validate that documentation doesn't reference deleted bd commands.
#
# Greps the docs/ tree, README, AGENTS.md, AGENT_INSTRUCTIONS.md, the integrations
# tree, and the Claude plugin for references to commands that were removed during
# the beads -> bd simplification.
#
# Usage: ./scripts/check-doc-flags.sh [bd-binary]
#
# Exit codes:
#   0 - Docs are consistent with the current CLI
#   1 - Stale references found

set -euo pipefail

BD="${1:-bd}"
ERRORS=0
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

if ! command -v "$BD" &>/dev/null && [ ! -x "$BD" ]; then
    echo "Error: bd binary not found at '$BD'"
    echo "Usage: $0 [path-to-bd]"
    exit 1
fi

echo "Checking documentation against CLI..."
echo "Using: $($BD version 2>/dev/null | head -1 || echo "$BD")"
echo ""

# Files to scan. Globs that don't match are silently dropped.
shopt -s nullglob globstar
DOC_FILES=(
    "$PROJECT_ROOT"/README.md
    "$PROJECT_ROOT"/AGENTS.md
    "$PROJECT_ROOT"/AGENT_INSTRUCTIONS.md
    "$PROJECT_ROOT"/docs/*.md
    "$PROJECT_ROOT"/docs/**/*.md
    "$PROJECT_ROOT"/integrations/**/*.md
    "$PROJECT_ROOT"/claude-plugin/**/*.md
)

# Commands that were deleted during the bd simplification. Anything in here
# should never appear in user-facing docs as a live command.
REMOVED_COMMANDS=(
    "bd hooks"
    "bd doctor"
    "bd quickstart"
    "bd export"
    "bd import"
    "bd info"
    "bd ship"
    "bd setup"
    "bd mol"
    "bd merge-slot"
    "bd rules"
    "bd kv"
    "bd todo"
    "bd link"
    "bd sync"
    "bd history"
    "bd diff"
    "bd branch"
)

for cmd in "${REMOVED_COMMANDS[@]}"; do
    REFS=$(grep -nE "\b${cmd}\b" "${DOC_FILES[@]}" 2>/dev/null \
        | grep -v 'CHANGELOG\|removed\|was removed\|has been removed\|no longer\|deprecated\|REMOVED\|<= v1.0' \
        || true)
    if [ -n "$REFS" ]; then
        echo "FAIL: stale references to removed '$cmd':"
        echo "$REFS" | head -10
        ERRORS=$((ERRORS + 1))
    fi
done

if [ $ERRORS -eq 0 ]; then
    echo "PASS: No stale references to removed commands"
fi

echo ""
echo "=== Summary ==="
if [ $ERRORS -gt 0 ]; then
    echo "FAILED: $ERRORS removed-command(s) referenced in docs"
    exit 1
fi
echo "PASSED: docs consistent with current CLI"
exit 0

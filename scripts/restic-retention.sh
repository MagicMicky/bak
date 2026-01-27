#!/bin/bash
#
# restic-retention.sh - Enforce retention policies based on client-declared tags
#
# Environment variables (set via systemd):
#   RESTIC_REPOSITORY     - Path to restic repository
#   RESTIC_PASSWORD_FILE  - Path to password file
#
# Usage:
#   restic-retention.sh [--dry-run]
#
# Options:
#   --dry-run    Show what would be deleted without actually deleting
#

set -euo pipefail

DRY_RUN=""
if [[ "${1:-}" == "--dry-run" ]]; then
    DRY_RUN="--dry-run"
    echo "*** DRY RUN MODE - no changes will be made ***"
    echo ""
fi

# Validate environment
if [[ -z "${RESTIC_REPOSITORY:-}" ]]; then
    echo "ERROR: RESTIC_REPOSITORY environment variable is not set" >&2
    exit 1
fi

if [[ -z "${RESTIC_PASSWORD_FILE:-}" ]]; then
    echo "ERROR: RESTIC_PASSWORD_FILE environment variable is not set" >&2
    exit 1
fi

echo "=============================================="
echo "Retention run: $(date)"
echo "Repository: $RESTIC_REPOSITORY"
[[ -n "$DRY_RUN" ]] && echo "Mode: DRY RUN"
echo "=============================================="

# Get all unique primary tags (excluding retain:* tags)
TAGS=$(restic snapshots --json 2>/dev/null | \
    jq -r '.[].tags[]? // empty' | \
    grep -v '^retain:' | \
    sort -u)

if [[ -z "$TAGS" ]]; then
    echo "No snapshots found."
    exit 0
fi

for TAG in $TAGS; do
    echo ""
    echo "--- $TAG ---"

    # Get retention policy from most recent snapshot's retain: tag
    RETAIN=$(restic snapshots --tag "$TAG" --json 2>/dev/null | \
        jq -r 'sort_by(.time) | last | .tags[]? // empty | select(startswith("retain:"))' | \
        sed 's/retain://' || echo "")

    if [[ -z "$RETAIN" ]]; then
        echo "  No policy found, using defaults"
        RETAIN="h=0,d=7,w=4,m=6,y=0"
    fi

    # Parse retention values
    H=$(echo "$RETAIN" | grep -oP 'h=\K\d+' || echo 0)
    D=$(echo "$RETAIN" | grep -oP 'd=\K\d+' || echo 7)
    W=$(echo "$RETAIN" | grep -oP 'w=\K\d+' || echo 4)
    M=$(echo "$RETAIN" | grep -oP 'm=\K\d+' || echo 6)
    Y=$(echo "$RETAIN" | grep -oP 'y=\K\d+' || echo 0)

    echo "  Retention: h=$H d=$D w=$W m=$M y=$Y"

    # Build forget arguments
    ARGS="--tag $TAG"
    [[ $H -gt 0 ]] && ARGS="$ARGS --keep-hourly $H"
    [[ $D -gt 0 ]] && ARGS="$ARGS --keep-daily $D"
    [[ $W -gt 0 ]] && ARGS="$ARGS --keep-weekly $W"
    [[ $M -gt 0 ]] && ARGS="$ARGS --keep-monthly $M"
    [[ $Y -gt 0 ]] && ARGS="$ARGS --keep-yearly $Y"

    restic forget $ARGS $DRY_RUN
done

echo ""
echo "--- Pruning ---"
if [[ -n "$DRY_RUN" ]]; then
    echo "  Skipped (dry run)"
else
    restic prune
fi

echo ""
echo "=============================================="
echo "Done: $(date)"
echo "=============================================="

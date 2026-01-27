#!/bin/bash
set -euo pipefail

# Restic Retention Script
# Applies per-tag retention policies based on retain: tags in snapshots
#
# Required environment variables:
#   RESTIC_REPOSITORY    - Path or URL to the restic repository
#   RESTIC_PASSWORD_FILE - Path to the file containing the repository password
#
# Optional environment variables:
#   LOG_FILE             - Path to log file (default: /var/log/restic-retention.log)
#
# Usage:
#   export RESTIC_REPOSITORY="/backups/myrepo"
#   export RESTIC_PASSWORD_FILE="/etc/restic/myrepo.password"
#   ./restic-retention.sh

# Validate required environment variables
if [[ -z "${RESTIC_REPOSITORY:-}" ]]; then
    echo "ERROR: RESTIC_REPOSITORY environment variable is not set" >&2
    exit 1
fi

if [[ -z "${RESTIC_PASSWORD_FILE:-}" ]]; then
    echo "ERROR: RESTIC_PASSWORD_FILE environment variable is not set" >&2
    exit 1
fi

if [[ ! -f "$RESTIC_PASSWORD_FILE" ]]; then
    echo "ERROR: Password file does not exist: $RESTIC_PASSWORD_FILE" >&2
    exit 1
fi

export RESTIC_REPOSITORY
export RESTIC_PASSWORD_FILE

LOG_FILE="${LOG_FILE:-/var/log/restic-retention.log}"
exec > >(tee -a "$LOG_FILE") 2>&1

echo "=============================================="
echo "Retention run: $(date)"
echo "=============================================="

# Get all unique primary tags
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

    # Get retention from latest snapshot's retain: tag
    RETAIN=$(restic snapshots --tag "$TAG" --json 2>/dev/null | \
        jq -r 'sort_by(.time) | last | .tags[]? // empty | select(startswith("retain:"))' | \
        sed 's/retain://' || echo "")

    if [[ -z "$RETAIN" ]]; then
        echo "  No policy, using defaults"
        RETAIN="h=0,d=7,w=4,m=6,y=0"
    fi

    H=$(echo "$RETAIN" | grep -oP 'h=\K\d+' || echo 0)
    D=$(echo "$RETAIN" | grep -oP 'd=\K\d+' || echo 7)
    W=$(echo "$RETAIN" | grep -oP 'w=\K\d+' || echo 4)
    M=$(echo "$RETAIN" | grep -oP 'm=\K\d+' || echo 6)
    Y=$(echo "$RETAIN" | grep -oP 'y=\K\d+' || echo 0)

    echo "  Retention: h=$H d=$D w=$W m=$M y=$Y"

    ARGS="--tag $TAG"
    [[ $H -gt 0 ]] && ARGS="$ARGS --keep-hourly $H"
    [[ $D -gt 0 ]] && ARGS="$ARGS --keep-daily $D"
    [[ $W -gt 0 ]] && ARGS="$ARGS --keep-weekly $W"
    [[ $M -gt 0 ]] && ARGS="$ARGS --keep-monthly $M"
    [[ $Y -gt 0 ]] && ARGS="$ARGS --keep-yearly $Y"

    restic forget $ARGS
done

echo ""
echo "--- Pruning ---"
restic prune

echo ""
echo "Done: $(date)"

#!/usr/bin/env bash
# Retain the active binary plus the newest rollback binaries in /opt/sub2api.
# Run only after a release has started successfully and no upload is in flight.

set -euo pipefail

INSTALL_DIR="${SUB2API_INSTALL_DIR:-/opt/sub2api}"
RETENTION="${SUB2API_BACKUP_RETENTION:-2}"

if [[ "$INSTALL_DIR" != /* || "$INSTALL_DIR" == "/" || ! -d "$INSTALL_DIR" ]]; then
    printf 'Invalid SUB2API_INSTALL_DIR: %s\n' "$INSTALL_DIR" >&2
    exit 2
fi

if ! [[ "$RETENTION" =~ ^[1-9][0-9]*$ ]]; then
    printf 'SUB2API_BACKUP_RETENTION must be a positive integer\n' >&2
    exit 2
fi

while IFS= read -r old_backup; do
    [[ -n "$old_backup" ]] || continue
    rm -f -- "$old_backup"
    printf 'Removed old binary backup: %s\n' "$old_backup"
done < <(
    find "$INSTALL_DIR" -maxdepth 1 -type f \
        \( -name 'sub2api.backup' -o -name 'sub2api.backup.*' -o -name 'sub2api.before-*' -o -name 'sub2api.rollback-*' \) \
        -printf '%T@ %p\n' |
        sort -nr |
        awk -v keep="$RETENTION" 'NR > keep { sub(/^[^ ]+ /, ""); print }'
)

# Completed releases do not need staged archives or interrupted upload parts.
find "$INSTALL_DIR" -maxdepth 1 -type f \
    \( -name 'sub2api.next-*' -o -name '.sub2api.*.part' \) -delete
find "$INSTALL_DIR" -maxdepth 1 -type d \
    \( -name 'stage-*' -o -name 'sub2api.stage-*' \) -empty -delete

printf 'Rollback binary retention: %s\n' "$RETENTION"

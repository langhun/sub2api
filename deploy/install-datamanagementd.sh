#!/usr/bin/env bash

set -euo pipefail

cat >&2 <<'EOF'
datamanagementd has been removed from current Sub2API releases.

This installer is kept only to prevent accidental use of obsolete deployment
instructions. Do not install sub2api-datamanagementd.service or mount
/tmp/sub2api-datamanagement.sock for current builds.

If an older server still has this service installed, disable it with:
  sudo systemctl disable --now sub2api-datamanagementd || true
  sudo rm -f /etc/systemd/system/sub2api-datamanagementd.service
  sudo systemctl daemon-reload
EOF

exit 1

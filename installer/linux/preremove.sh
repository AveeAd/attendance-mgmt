#!/usr/bin/env bash
set -euo pipefail

if systemctl is-enabled --quiet attendance-mgmt 2>/dev/null || systemctl is-active --quiet attendance-mgmt 2>/dev/null; then
  systemctl disable --now attendance-mgmt
fi

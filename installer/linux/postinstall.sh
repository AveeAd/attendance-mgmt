#!/usr/bin/env bash
set -euo pipefail

if ! id attendance-mgmt >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /usr/sbin/nologin attendance-mgmt
fi

mkdir -p /var/lib/attendance-mgmt
chown attendance-mgmt:attendance-mgmt /var/lib/attendance-mgmt
chmod 0750 /var/lib/attendance-mgmt

systemctl daemon-reload
systemctl enable --now attendance-mgmt

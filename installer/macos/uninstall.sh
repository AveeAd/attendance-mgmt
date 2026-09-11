#!/usr/bin/env bash
# Manual uninstaller — macOS .pkg installers have no built-in uninstall.
# Run with: sudo ./uninstall.sh
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "Run this with sudo." >&2
  exit 1
fi

CONSOLE_USER=$(stat -f%Su /dev/console)
USER_HOME=$(dscl . -read "/Users/${CONSOLE_USER}" NFSHomeDirectory | awk '{print $2}')
PLIST="${USER_HOME}/Library/LaunchAgents/com.attendance-mgmt.app.plist"

if [ -f "$PLIST" ]; then
  sudo -u "${CONSOLE_USER}" launchctl unload "$PLIST" >/dev/null 2>&1 || true
  rm -f "$PLIST"
fi

rm -rf /usr/local/attendance-mgmt

echo "Uninstalled. Your attendance data was left in place at:"
echo "  ${USER_HOME}/Library/Application Support/attendance-mgmt"
echo "Delete that folder yourself if you want to remove it too."

#!/usr/bin/env bash
set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVER_BIN="$PROJECT_DIR/mac_arm/server"
CLI_BIN="$PROJECT_DIR/mac_arm/cli"

PLIST_DIR="$HOME/Library/LaunchAgents"
PLIST_NAME="com.ipmsg.server.plist"
PLIST_FILE="$PLIST_DIR/$PLIST_NAME"

echo "[*] Checking binaries..."
[[ -x "$SERVER_BIN" ]] || { echo "❌ server binary not found or not executable"; exit 1; }
[[ -x "$CLI_BIN" ]] || { echo "❌ cli binary not found or not executable"; exit 1; }

echo "[*] Creating LaunchAgent..."
mkdir -p "$PLIST_DIR"

cat > "$PLIST_FILE" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.ipmsg.server</string>

    <key>ProgramArguments</key>
    <array>
        <string>$SERVER_BIN</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
EOF

echo "[*] Loading LaunchAgent..."
launchctl unload "$PLIST_FILE" 2>/dev/null || true
launchctl load "$PLIST_FILE"

echo "[*] Installing CLI as ipmsg..."
if [[ ! -w /usr/local/bin ]]; then
    echo "[!] sudo required for /usr/local/bin"
    sudo ln -sf "$CLI_BIN" /usr/local/bin/ipmsg
else
    ln -sf "$CLI_BIN" /usr/local/bin/ipmsg
fi

echo "✅ Installation completed successfully!"
echo "  • server is running in background and starts on login"
echo "  • cli is available as: ipmsg"

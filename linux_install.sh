#!/usr/bin/env bash
set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVER_BIN="$PROJECT_DIR/linux/server"
CLI_BIN="$PROJECT_DIR/linux/cli"

SERVICE_NAME="ipmsg-server"
SERVICE_DIR="$HOME/.config/systemd/user"
SERVICE_FILE="$SERVICE_DIR/$SERVICE_NAME.service"

echo "[*] Checking binaries..."
[[ -x "$SERVER_BIN" ]] || { echo "❌ $SERVER_BIN not found or not executable"; exit 1; }
[[ -x "$CLI_BIN" ]] || { echo "❌ $CLI_BIN not found or not executable"; exit 1; }

echo "[*] Creating systemd user service..."
mkdir -p "$SERVICE_DIR"

cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=IPMSG Server
After=network.target

[Service]
ExecStart=$SERVER_BIN
Restart=always
RestartSec=3

[Install]
WantedBy=default.target
EOF

echo "[*] Reloading systemd user daemon..."
systemctl --user daemon-reexec
systemctl --user daemon-reload

echo "[*] Enabling service at startup and starting it now..."
systemctl --user enable --now "$SERVICE_NAME"

echo "[*] Adding CLI binary to PATH as 'ipmsg'..."
if [[ ! -w /usr/local/bin ]]; then
    echo "[!] sudo required to write into /usr/local/bin"
    sudo ln -sf "$CLI_BIN" /usr/local/bin/ipmsg
else
    ln -sf "$CLI_BIN" /usr/local/bin/ipmsg
fi

echo "✅ Installation completed successfully!"
echo "  • server is running in background and enabled at startup"
echo "  • cli is available globally as: ipmsg"

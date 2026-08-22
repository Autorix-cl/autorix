#!/usr/bin/env bash
set -e

# Autorix Engine Linux Installer (Bare-metal)
# Usage: curl -sL https://autorix.com/install.sh | sudo bash -s -- --engine <engine_name>

ENGINE=""
VERSION="latest"

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --engine) ENGINE="$2"; shift ;;
        --version) VERSION="$2"; shift ;;
        *) echo "Unknown parameter passed: $1"; exit 1 ;;
    esac
    shift
done

VALID_ENGINES=("argus" "nexus" "ego" "janus" "aegis" "vulcan" "hermes" "themis")

if [[ ! " ${VALID_ENGINES[@]} " =~ " ${ENGINE} " ]]; then
    echo "ERROR: You must specify a valid engine to install."
    echo "Valid engines: ${VALID_ENGINES[*]}"
    echo "Usage: curl -sL https://.../install.sh | sudo bash -s -- --engine <name>"
    exit 1
fi

echo "🚀 Installing Autorix Engine: $ENGINE ($VERSION)..."

# 1. Create Autorix system user
if ! id -u autorix > /dev/null 2>&1; then
    echo "=> Creating 'autorix' system user..."
    useradd --system --shell /bin/false --no-create-home autorix
fi

# 2. Setup directories
echo "=> Setting up directories in /etc/autorix and /var/lib/autorix..."
mkdir -p /etc/autorix
mkdir -p /var/lib/autorix/$ENGINE
chown -R autorix:autorix /etc/autorix /var/lib/autorix

# 3. Download Binary
BIN_URL="https://github.com/Autorix-cl/autorix/releases/download/${VERSION}/${ENGINE}d-linux-amd64"
echo "=> Downloading ${ENGINE}d from $BIN_URL..."
# curl -sL -o /usr/local/bin/autorix-$ENGINE "$BIN_URL"
# chmod +x /usr/local/bin/autorix-$ENGINE
echo "   (Binary download mocked for now until GitHub Releases are active)"

# 4. Generate Environment File
ENV_FILE="/etc/autorix/${ENGINE}.env"
if [ ! -f "$ENV_FILE" ]; then
    echo "=> Generating environment config at $ENV_FILE..."
    cat <<EOF > "$ENV_FILE"
# Autorix $ENGINE Configuration
AUTORIX_ENV=production
# Update this with your actual PostgreSQL connection string
POSTGRES_DSN=postgres://autorix:password@localhost:5432/autorix_${ENGINE}?sslmode=disable
EOF

    # If it's not the control plane (Argus), it needs to know how to connect to Argus
    if [ "$ENGINE" != "argus" ]; then
        echo "AUTORIX_ARGUS_URL=http://localhost:4400" >> "$ENV_FILE"
        echo "AUTORIX_ENROLLMENT_TOKEN=aet_..." >> "$ENV_FILE"
    fi
    
    chown autorix:autorix "$ENV_FILE"
    chmod 600 "$ENV_FILE"
fi

# 5. Create Systemd Service
SERVICE_FILE="/etc/systemd/system/autorix-${ENGINE}.service"
echo "=> Creating systemd unit at $SERVICE_FILE..."
cat <<EOF > "$SERVICE_FILE"
[Unit]
Description=Autorix Engine - $ENGINE
Documentation=https://github.com/Autorix-cl/autorix/tree/main/docs
After=network.target

[Service]
Type=simple
User=autorix
Group=autorix
EnvironmentFile=$ENV_FILE
ExecStart=/usr/local/bin/autorix-${ENGINE}
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

# 6. Enable (but don't start automatically since env vars need setup)
echo "=> Reloading systemd daemon..."
systemctl daemon-reload
systemctl enable autorix-${ENGINE}.service

echo ""
echo "✅ Installation complete for Autorix $ENGINE!"
echo "---------------------------------------------------"
echo "Next steps:"
echo "1. Edit the database config:  nano $ENV_FILE"
echo "2. Start the engine:          systemctl start autorix-${ENGINE}.service"
echo "3. View the logs:             journalctl -fu autorix-${ENGINE}.service"
echo "---------------------------------------------------"

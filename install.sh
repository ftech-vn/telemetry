#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO="ftech-vn/telemetry"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="telemetry"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Convert arch to Go naming convention
case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Convert OS to Go naming convention
case $OS in
    darwin)
        PLATFORM="darwin"
        ;;
    linux)
        PLATFORM="linux"
        ;;
    mingw*|msys*|cygwin*)
        PLATFORM="windows"
        BINARY_NAME="telemetry.exe"
        ;;
    *)
        echo -e "${RED}Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

BINARY_FILE="${BINARY_NAME%.*}-${PLATFORM}-${ARCH}"
if [ "$PLATFORM" = "windows" ]; then
    BINARY_FILE="${BINARY_FILE}.exe"
fi

# Parse arguments
AUTO_START=false
while [[ $# -gt 0 ]]; do
  case $1 in
    --server-id)
      ARG_SERVER_ID="$2"
      shift 2
      ;;
    --server-key)
      ARG_SERVER_KEY="$2"
      shift 2
      ;;
    --webhook-url)
      ARG_WEBHOOK_URL="$2"
      shift 2
      ;;
    --server-name)
      ARG_SERVER_NAME="$2"
      shift 2
      ;;
    --cpu-threshold)
      ARG_CPU_THRESHOLD="$2"
      shift 2
      ;;
    --memory-threshold)
      ARG_MEMORY_THRESHOLD="$2"
      shift 2
      ;;
    --disk-threshold)
      ARG_DISK_THRESHOLD="$2"
      shift 2
      ;;
    --auto-start)
      AUTO_START=true
      shift
      ;;
    *)
      shift
      ;;
  esac
done

echo -e "${GREEN}Installing telemetry for ${PLATFORM}-${ARCH}...${NC}"

# Get latest release info from GitHub
echo -e "${YELLOW}Fetching latest release...${NC}"
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest")
DOWNLOAD_URL=$(echo "$LATEST_RELEASE" | grep "\"browser_download_url\": \".*${BINARY_FILE}\"" | cut -d '"' -f 4)

if [ -z "$DOWNLOAD_URL" ]; then
    echo -e "${RED}Could not find release for ${PLATFORM}-${ARCH}${NC}"
    echo -e "${YELLOW}Falling back to building from source...${NC}"
    
    # Check for Go and Git
    if ! command -v go &> /dev/null || ! command -v git &> /dev/null; then
        echo -e "${RED}Go and Git are required to build from source.${NC}"
        echo -e "${YELLOW}Please install them and try again.${NC}"
        exit 1
    fi

    # Build from source
    echo -e "${YELLOW}Building from source...${NC}"
    TMP_DIR=$(mktemp -d)
    git clone --depth 1 "https://github.com/${REPO}.git" "$TMP_DIR"
    cd "$TMP_DIR"
    go build -o "${BINARY_NAME}" .
    cd - > /dev/null
    TMP_FILE="${TMP_DIR}/${BINARY_NAME}"

else
    # Download binary
    echo -e "${YELLOW}Downloading from: $DOWNLOAD_URL${NC}"
    TMP_FILE="/tmp/${BINARY_NAME}"
    curl -L -o "$TMP_FILE" "$DOWNLOAD_URL"
fi

# Make executable
chmod +x "$TMP_FILE"

# Install to /usr/local/bin (requires sudo on most systems)
echo -e "${YELLOW}Installing to ${INSTALL_DIR}...${NC}"
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
    echo -e "${GREEN}✓ Binary installed to ${INSTALL_DIR}/${BINARY_NAME}${NC}"
elif command -v sudo &> /dev/null && sudo -n true 2>/dev/null; then
    # sudo available and passwordless
    sudo mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
    echo -e "${GREEN}✓ Binary installed to ${INSTALL_DIR}/${BINARY_NAME}${NC}"
else
    # Need sudo with password
    echo -e "${YELLOW}Installation requires sudo access.${NC}"
    echo -e "Please enter your password when prompted:"
    if sudo mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"; then
        echo -e "${GREEN}✓ Binary installed to ${INSTALL_DIR}/${BINARY_NAME}${NC}"
    else
        echo -e "${RED}Failed to install binary. You can manually move it:${NC}"
        echo -e "  ${YELLOW}sudo mv $TMP_FILE ${INSTALL_DIR}/${BINARY_NAME}${NC}"
        exit 1
    fi
fi

# Create config directory and default config
CONFIG_DIR="$HOME/.telemetry"
CONFIG_FILE="$CONFIG_DIR/config.yaml"

echo -e "${YELLOW}Updating configuration...${NC}"
mkdir -p "$CONFIG_DIR"

# Function to add config if missing
add_config_if_missing() {
    local key=$1
    local value=$2
    if ! grep -q "^${key}:" "$CONFIG_FILE" 2>/dev/null; then
        echo "${key}: ${value}" >> "$CONFIG_FILE"
        return 0
    fi
    return 1
}

if [ ! -f "$CONFIG_FILE" ]; then
    cat > "$CONFIG_FILE" << EOF
# Telemetry Configuration

# Auto-update feature
auto_update: false

# Server identification
server_name: "${ARG_SERVER_NAME:-"production-server-1"}"
webhook_url: "${ARG_WEBHOOK_URL:-""}"
webhook_interval: "1s"
gemini_webhook_url: ""
server_id: "${ARG_SERVER_ID:-""}"
server_key: "${ARG_SERVER_KEY:-""}"

# Lark webhook URL (optional)
lark_webhook_url: "https://open.larksuite.com/open-apis/bot/v2/hook/your-webhook-here"

# Check interval - how often to monitor metrics
check_interval: "60s"

# Disk usage threshold percentage - alert when disk usage exceeds this value
disk_threshold: ${ARG_DISK_THRESHOLD:-80.0}

# CPU usage threshold percentage - alert when CPU usage exceeds this
cpu_threshold: ${ARG_CPU_THRESHOLD:-80.0}

# Memory usage threshold percentage - alert when memory usage exceeds this
memory_threshold: ${ARG_MEMORY_THRESHOLD:-80.0}

# Directories to exclude from disk breakdown (optional)
excluded_dirs: []

# Health check URLs (optional)
health_checks: []

# Database connection checks (optional)
db_checks: []
EOF
    echo -e "${GREEN}✓ Created config file at ${CONFIG_FILE}${NC}"
else
    echo -e "${YELLOW}Config file already exists. Updating fields...${NC}"
    CHANGES=0
    
    # Function to update or add config
    update_config() {
        local key=$1
        local value=$2
        if grep -q "^${key}:" "$CONFIG_FILE"; then
            # Use sed to replace the line
            # Careful with slashes in value, use a different delimiter for sed like |
            sed -i.bak "s|^${key}:.*|${key}: ${value}|" "$CONFIG_FILE"
            rm -f "${CONFIG_FILE}.bak"
        else
            echo "${key}: ${value}" >> "$CONFIG_FILE"
        fi
        CHANGES=$((CHANGES+1))
    }

    if [ -n "$ARG_SERVER_NAME" ]; then update_config "server_name" "\"${ARG_SERVER_NAME}\""; fi
    if [ -n "$ARG_WEBHOOK_URL" ]; then update_config "webhook_url" "\"${ARG_WEBHOOK_URL}\""; fi
    if [ -n "$ARG_SERVER_ID" ]; then update_config "server_id" "\"${ARG_SERVER_ID}\""; fi
    if [ -n "$ARG_SERVER_KEY" ]; then update_config "server_key" "\"${ARG_SERVER_KEY}\""; fi
    if [ -n "$ARG_CPU_THRESHOLD" ]; then update_config "cpu_threshold" "${ARG_CPU_THRESHOLD}"; fi
    if [ -n "$ARG_MEMORY_THRESHOLD" ]; then update_config "memory_threshold" "${ARG_MEMORY_THRESHOLD}"; fi
    if [ -n "$ARG_DISK_THRESHOLD" ]; then update_config "disk_threshold" "${ARG_DISK_THRESHOLD}"; fi

    add_config_if_missing "lark_webhook_url" "\"https://open.larksuite.com/open-apis/bot/v2/hook/your-webhook-here\"" && CHANGES=$((CHANGES+1))
    add_config_if_missing "check_interval" "\"60s\"" && CHANGES=$((CHANGES+1))
    add_config_if_missing "disk_threshold" "${ARG_DISK_THRESHOLD:-80.0}" && CHANGES=$((CHANGES+1))
    add_config_if_missing "cpu_threshold" "${ARG_CPU_THRESHOLD:-80.0}" && CHANGES=$((CHANGES+1))
    add_config_if_missing "memory_threshold" "${ARG_MEMORY_THRESHOLD:-80.0}" && CHANGES=$((CHANGES+1))
    add_config_if_missing "health_checks" "[]" && CHANGES=$((CHANGES+1))
    add_config_if_missing "db_checks" "[]" && CHANGES=$((CHANGES+1))
    add_config_if_missing "excluded_dirs" "[]" && CHANGES=$((CHANGES+1))
    add_config_if_missing "webhook_interval" "\"30s\"" && CHANGES=$((CHANGES+1))
    add_config_if_missing "auto_update" "false" && CHANGES=$((CHANGES+1))
    add_config_if_missing "gemini_api_key" "\"\"" && CHANGES=$((CHANGES+1))
    add_config_if_missing "gemini_webhook_url" "\"\"" && CHANGES=$((CHANGES+1))
    
    if [ $CHANGES -gt 0 ]; then
        echo -e "${GREEN}✓ Added $CHANGES new configuration fields to ${CONFIG_FILE}${NC}"
    else
        echo -e "${GREEN}✓ Configuration is up to date${NC}"
    fi
fi

# Verify installation
if command -v "$BINARY_NAME" &> /dev/null; then
    echo -e "${GREEN}✓ Successfully installed ${BINARY_NAME}!${NC}"
    echo -e ""
    echo -e "${GREEN}Next steps:${NC}"
    echo -e "  1. Edit config: ${YELLOW}${CONFIG_FILE}${NC}"
    echo -e "  2. Set your Lark webhook URL"
    echo -e ""
    echo -e "${YELLOW}Setting up system service...${NC}"
    
    # Setup system service based on OS
    case $PLATFORM in
        darwin)
            # macOS - launchd
            PLIST_FILE="$HOME/Library/LaunchAgents/com.telemetry.monitor.plist"
            mkdir -p "$HOME/Library/LaunchAgents"
            
            cat > "$PLIST_FILE" << PLIST_EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.telemetry.monitor</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/${BINARY_NAME}</string>
        <string>run</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${HOME}/.telemetry/telemetry.log</string>
    <key>StandardErrorPath</key>
    <string>${HOME}/.telemetry/telemetry.err</string>
</dict>
</plist>
PLIST_EOF
            
            if [ "$AUTO_START" = true ]; then
                echo -e "${GREEN}✓ Created launchd service${NC}"
                echo -e "${YELLOW}Starting service...${NC}"
                launchctl load "$PLIST_FILE"
                echo -e "  Logs:  ${YELLOW}tail -f ~/.telemetry/telemetry.log${NC}"
            else
                echo -e "${GREEN}✓ Created launchd service (not started yet)${NC}"
                echo -e "  Start: ${YELLOW}launchctl load ~/Library/LaunchAgents/com.telemetry.monitor.plist${NC}"
                echo -e "  Stop:  ${YELLOW}launchctl unload ~/Library/LaunchAgents/com.telemetry.monitor.plist${NC}"
                echo -e "  Logs:  ${YELLOW}tail -f ~/.telemetry/telemetry.log${NC}"
            fi
            ;;
            
        linux)
            # Linux - systemd
            SERVICE_FILE="/etc/systemd/system/telemetry.service"
            
            cat > /tmp/telemetry.service << SERVICE_EOF
[Unit]
Description=Telemetry System Monitor
After=network.target

[Service]
Type=simple
User=${USER}
ExecStart=${INSTALL_DIR}/${BINARY_NAME} run
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=10
StandardOutput=append:${HOME}/.telemetry/telemetry.log
StandardError=append:${HOME}/.telemetry/telemetry.err

[Install]
WantedBy=multi-user.target
SERVICE_EOF
            
            # Check if we can write to systemd directory
            if [ -w /etc/systemd/system ] 2>/dev/null; then
                mv /tmp/telemetry.service "$SERVICE_FILE"
                systemctl daemon-reload 2>/dev/null || true
                if [ "$AUTO_START" = true ]; then
                    systemctl enable telemetry 2>/dev/null || true
                    systemctl start telemetry 2>/dev/null || true
                    echo -e "${GREEN}✓ Created and started systemd service${NC}"
                else
                    echo -e "${GREEN}✓ Created systemd service${NC}"
                fi
            elif command -v sudo &> /dev/null && sudo -n true 2>/dev/null; then
                # sudo available and passwordless
                sudo mv /tmp/telemetry.service "$SERVICE_FILE"
                sudo systemctl daemon-reload
                if [ "$AUTO_START" = true ]; then
                    sudo systemctl enable telemetry
                    sudo systemctl start telemetry
                    echo -e "${GREEN}✓ Created and started systemd service${NC}"
                else
                    echo -e "${GREEN}✓ Created systemd service${NC}"
                fi
            else
                # Need sudo with password - show manual steps
                if [ "$AUTO_START" = true ]; then
                    echo -e "${YELLOW}⚠️  Cannot auto-install systemd service (requires sudo)${NC}"
                    echo -e ""
                    echo -e "${YELLOW}Please run these commands to install and start the service:${NC}"
                    echo -e "  ${YELLOW}sudo mv /tmp/telemetry.service /etc/systemd/system/telemetry.service${NC}"
                    echo -e "  ${YELLOW}sudo systemctl daemon-reload${NC}"
                    echo -e "  ${YELLOW}sudo systemctl enable telemetry${NC}"
                    echo -e "  ${YELLOW}sudo systemctl start telemetry${NC}"
                    echo -e ""
                else
                    echo -e "${YELLOW}⚠️  Cannot auto-install systemd service (requires sudo)${NC}"
                    echo -e ""
                    echo -e "${YELLOW}Please run these commands to install the service:${NC}"
                    echo -e "  ${YELLOW}sudo mv /tmp/telemetry.service /etc/systemd/system/telemetry.service${NC}"
                    echo -e "  ${YELLOW}sudo systemctl daemon-reload${NC}"
                    echo -e ""
                    echo -e "Service file is ready at: ${YELLOW}/tmp/telemetry.service${NC}"
                fi
            fi
            
            if [ "$AUTO_START" != true ]; then
                echo -e ""
                echo -e "${GREEN}Service management:${NC}"
                echo -e "  Enable: ${YELLOW}sudo systemctl enable telemetry${NC}"
                echo -e "  Start:  ${YELLOW}sudo systemctl start telemetry${NC}"
                echo -e "  Stop:   ${YELLOW}sudo systemctl stop telemetry${NC}"
                echo -e "  Status: ${YELLOW}sudo systemctl status telemetry${NC}"
            fi
            
            echo -e "  Logs:   ${YELLOW}journalctl -u telemetry -f${NC}"
            ;;
            
        windows)
            # Windows - NSSM or manual service setup
            echo -e "${YELLOW}Windows service setup:${NC}"
            echo -e "  1. Download NSSM: ${YELLOW}https://nssm.cc/download${NC}"
            echo -e "  2. Run: ${YELLOW}nssm install telemetry \"${INSTALL_DIR}\\${BINARY_NAME}\"${NC}"
            echo -e "  3. Configure logging in NSSM GUI"
            echo -e ""
            echo -e "  Or use Task Scheduler:"
            echo -e "  ${YELLOW}schtasks /create /tn \"Telemetry\" /tr \"${INSTALL_DIR}\\${BINARY_NAME}\" /sc onstart /ru SYSTEM${NC}"
            ;;
    esac
    
    echo -e ""
    echo -e "${GREEN}Installation complete!${NC}"
    echo -e ""
    echo -e "${YELLOW}⚠️  IMPORTANT: Before starting the service:${NC}"
    echo -e "  1. Edit config: ${YELLOW}${CONFIG_FILE}${NC}"
    echo -e "  2. Set your Lark webhook URL"
    echo -e "  3. Then start the service using the commands above"
else
    echo -e "${RED}Installation completed but ${BINARY_NAME} is not in PATH${NC}"
    echo -e "${YELLOW}You may need to add ${INSTALL_DIR} to your PATH${NC}"
fi

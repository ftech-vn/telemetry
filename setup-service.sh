#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Setting up telemetry as a background service...${NC}"

# Check if running on macOS
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo -e "${YELLOW}Detected macOS - using launchd${NC}"
    
    PLIST_FILE="$HOME/Library/LaunchAgents/com.telemetry.monitor.plist"
    
    # Create LaunchAgents directory if it doesn't exist
    mkdir -p "$HOME/Library/LaunchAgents"
    
    # Copy the plist file
    cat > "$PLIST_FILE" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.telemetry.monitor</string>
    
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/telemetry</string>
    </array>
    
    <key>RunAtLoad</key>
    <true/>
    
    <key>KeepAlive</key>
    <true/>
    
    <key>StandardOutPath</key>
    <string>/tmp/telemetry.log</string>
    
    <key>StandardErrorPath</key>
    <string>/tmp/telemetry.err</string>
    
    <key>WorkingDirectory</key>
    <string>/tmp</string>
</dict>
</plist>
EOF
    
    # Load the service
    launchctl unload "$PLIST_FILE" 2>/dev/null || true
    launchctl load "$PLIST_FILE"
    
    echo -e "${GREEN}✓ Service installed and started!${NC}"
    echo -e ""
    echo -e "${GREEN}Commands:${NC}"
    echo -e "  View logs:    ${YELLOW}tail -f /tmp/telemetry.log${NC}"
    echo -e "  View errors:  ${YELLOW}tail -f /tmp/telemetry.err${NC}"
    echo -e "  Stop service: ${YELLOW}launchctl unload ~/Library/LaunchAgents/com.telemetry.monitor.plist${NC}"
    echo -e "  Start again:  ${YELLOW}launchctl load ~/Library/LaunchAgents/com.telemetry.monitor.plist${NC}"
    echo -e "  Check status: ${YELLOW}launchctl list | grep telemetry${NC}"
    
elif [[ -f /etc/systemd/system ]]; then
    echo -e "${YELLOW}Detected Linux - using systemd${NC}"
    
    SYSTEMD_FILE="/etc/systemd/system/telemetry.service"
    
    # Create systemd service file
    sudo bash -c "cat > $SYSTEMD_FILE" << EOF
[Unit]
Description=Telemetry System Monitor
After=network.target

[Service]
Type=simple
User=$USER
ExecStart=/usr/local/bin/telemetry
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
    
    # Reload systemd and enable service
    sudo systemctl daemon-reload
    sudo systemctl enable telemetry
    sudo systemctl start telemetry
    
    echo -e "${GREEN}✓ Service installed and started!${NC}"
    echo -e ""
    echo -e "${GREEN}Commands:${NC}"
    echo -e "  View logs:    ${YELLOW}journalctl -u telemetry -f${NC}"
    echo -e "  Stop service: ${YELLOW}sudo systemctl stop telemetry${NC}"
    echo -e "  Start again:  ${YELLOW}sudo systemctl start telemetry${NC}"
    echo -e "  Check status: ${YELLOW}sudo systemctl status telemetry${NC}"
    echo -e "  Disable:      ${YELLOW}sudo systemctl disable telemetry${NC}"
    
else
    echo -e "${RED}Unsupported OS. Manual setup required.${NC}"
    echo -e "${YELLOW}Use nohup: ${NC}nohup telemetry > ~/telemetry.log 2>&1 &"
    exit 1
fi

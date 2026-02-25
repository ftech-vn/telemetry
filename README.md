# Telemetry - System Monitoring & Alerting

A lightweight, extensible Go service that monitors system metrics (Disk, CPU, Memory, HTTP) and sends alerts to Lark.

## Quick Start

### Installation

**One-line install (recommended):**

```bash
curl -fsSL https://raw.githubusercontent.com/ftech-vn/telemetry/main/install.sh | bash
```

The script will detect your OS/architecture, download (or build) the binary, set up a default configuration, and install the system service.

## Features

- **Multi-Metric Monitoring**:
    - **Disk**: Monitors the root partition (`/`) with a detailed breakdown of top-level directory sizes.
    - **CPU**: Tracks total usage and provides a normalized list of the top 5 CPU-consuming processes.
    - **Memory**: Tracks memory usage (percentage and MB) with a list of top memory-consuming processes.
    - **HTTP Health**: Monitors multiple endpoints with custom project names.
    - **Database Connectivity**: Monitors full connectivity (including authentication) for Postgres and MySQL.
- **Intelligent Alerting**:
    - Single aggregated alert for disk usage instead of spamming per partition.
    - Top process breakdown to help you identify resource hogs instantly.
    - Automatic exclusion of virtual/ephemeral filesystems and tiny folders.
- **Zero-Downtime Reload**: Supports `systemctl reload` to apply configuration changes without restarting.
- **Flexible Configuration**: Disable any monitor by simply commenting out its threshold in the config.
- **Graceful Lifecycle**: Sends a notification when the service starts or shuts down (SIGTERM/SIGINT).
- **Multi-Platform**: Robust support for Linux, macOS, and Windows.

## Architecture

```
telemetry/
├── main.go                     # Entry point & Signal handling
├── install.sh                  # One-line install script
├── telemetry.service           # systemd unit template
└── internal/
    ├── config/                 # YAML-based configuration
    ├── monitor/                # Monitoring implementations
    │   ├── monitor.go          # Monitor interface & registry
    │   ├── cpu.go              # CPU usage & process tracking
    │   ├── memory.go           # Memory usage & process tracking
    │   ├── disk_unix.go        # Optimized Linux/macOS disk scanning
    │   ├── disk_windows.go     # Windows disk monitoring
    │   └── health.go           # HTTP endpoint health checks
    └── notifier/               # Notification implementations
        ├── notifier.go         # Notifier interface & registry
        └── lark.go             # Lark (ByteDance) notifier
```

## Configuration

The configuration is stored in `~/.telemetry/config.yaml`.

### Example Config

```yaml
# Server identification (appears in alerts)
server_name: "production-server-1"

# Lark webhook URL (required)
lark_webhook_url: "https://open.larksuite.com/open-apis/bot/v2/hook/xxx"

# Check interval (e.g., "30s", "1m", "5m")
check_interval: "60s"

# Thresholds (Comment out to disable a monitor)
disk_threshold: 80.0
excluded_dirs:
  - "/var/lib/docker"
  - "/tmp"

cpu_threshold: 80.0
memory_threshold: 80.0

# Multi-project HTTP Health Checks
# Format: "Project Name;URL"
health_checks:
  - "Main Website;https://example.com/health"
  - "API Gateway;https://api.example.com/ping"

# Database Connection Checks
# Format: "Name;DSN"
db_checks:
  - "Production DB;postgres://user:password@localhost:5432/dbname?sslmode=disable"
  - "Local DB;user:password@tcp(127.0.0.1:3306)/dbname"
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `server_name` | String | `unknown` | How this server identifies itself in alerts. |
| `lark_webhook_url`| String | (Required) | Your Lark bot webhook URL. |
| `check_interval` | Duration| `60s` | Frequency of checks (e.g. `10s`, `1m`). |
| `disk_threshold` | Float | `80.0` | Alert if root disk usage % exceeds this. |
| `excluded_dirs`  | List | `[]` | List of folders/paths to ignore in disk details. |
| `cpu_threshold`  | Float | `80.0` | Alert if total CPU usage % exceeds this. |
| `memory_threshold`| Float | `80.0` | Alert if memory usage % exceeds this. |
| `health_checks`  | List | `[]` | List of `Name;URL` to monitor for HTTP 200. |
| `db_checks`      | List | `[]` | List of `Name;DSN` for DB connectivity. |

## Service Management

### Linux (systemd)
```bash
# Start the service
sudo systemctl start telemetry

# Reload configuration (Zero-downtime)
sudo systemctl reload telemetry

# Restart the service
sudo systemctl restart telemetry

# View live logs
journalctl -u telemetry -f

# Stop the service
sudo systemctl stop telemetry

# Delete/Uninstall the service
sudo systemctl disable telemetry
sudo rm /etc/systemd/system/telemetry.service
sudo systemctl daemon-reload
```

### macOS (launchd)
```bash
# Start the service
launchctl load ~/Library/LaunchAgents/com.telemetry.monitor.plist

# Restart/Reload configuration
launchctl unload ~/Library/LaunchAgents/com.telemetry.monitor.plist
launchctl load ~/Library/LaunchAgents/com.telemetry.monitor.plist

# View logs
tail -f ~/.telemetry/telemetry.log

# Stop the service
launchctl unload ~/Library/LaunchAgents/com.telemetry.monitor.plist

# Delete/Uninstall the service
rm ~/Library/LaunchAgents/com.telemetry.monitor.plist
```

### Windows (NSSM)
```powershell
# Start the service
nssm start telemetry

# Restart the service
nssm restart telemetry

# Stop the service
nssm stop telemetry

# Delete/Uninstall the service
nssm remove telemetry confirm

# View logs
Get-Content -Path "$HOME\.telemetry\telemetry.log" -Wait
```

## Development

```bash
# Build locally
go build -o telemetry

# Run tests
go test ./...

# Create a new release (triggers GitHub Actions)
git tag v0.1.x
git push origin v0.1.x
```

## License

MIT

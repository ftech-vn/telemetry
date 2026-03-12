package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"telemetry/internal/config"
	"telemetry/internal/monitor"
	"telemetry/internal/notifier"
	"telemetry/internal/updater"
)

var Version = "dev"

func main() {
	log.Println("🚀 Telemetry service starting...")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Initial configuration and startup
	ctx, cancel := context.WithCancel(context.Background())
	cfg, monitors, notifiers, ticker := startup(ctx)
	defer ticker.Stop()
	defer cancel()

	// Start auto-updater
	go func() {
		if !cfg.AutoUpdate {
			return
		}

		// Calculate duration until next midnight
		now := time.Now()
		midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		durationUntilMidnight := midnight.Sub(now)

		log.Printf("🕒 Next auto-update check scheduled in %v (at %v)", durationUntilMidnight, midnight)

		// Wait until the first midnight
		firstCheckTimer := time.NewTimer(durationUntilMidnight)
		defer firstCheckTimer.Stop()

		select {
		case <-firstCheckTimer.C:
			// Run the first check
			updater.CheckForUpdates(Version, cfg)

			// Then, start a ticker for every 24 hours
			updateTicker := time.NewTicker(24 * time.Hour)
			defer updateTicker.Stop()

			for {
				select {
				case <-updateTicker.C:
					updater.CheckForUpdates(Version, cfg)
				case <-ctx.Done():
					return
				}
			}
		case <-ctx.Done():
			// If context is cancelled before the first check
			return
		}
	}()

	// Run initial check
	runAlertChecks(monitors, notifiers)

	for {
		select {
		case <-ticker.C:
			runAlertChecks(monitors, notifiers)
		case sig := <-sigChan:
			if sig == syscall.SIGHUP {
				log.Println("♻️  Received SIGHUP, reloading configuration...")
				cancel() // Stop existing TCP goroutines
				ctx, cancel = context.WithCancel(context.Background())
				newCfg, newMonitors, newNotifiers, newTicker := startup(ctx)
				if newTicker != nil {
					ticker.Stop()
					cfg, monitors, notifiers, ticker = newCfg, newMonitors, newNotifiers, newTicker
					log.Println("✅ Configuration reloaded successfully")
					runAlertChecks(monitors, notifiers)
				}
				continue
			}

			log.Printf("⚠️  Received signal %v, shutting down gracefully...", sig)
			
			// Notify about shutdown
			shutdownAlert := []monitor.Alert{
				{
					ServerName: cfg.ServerName,
					Type:       "system",
					Message:    "🛑 Telemetry service is shutting down...",
					Severity:   "warning",
				},
			}
			notifiers.NotifyAll(shutdownAlert)

			log.Println("🛑 Stopping ticker...")
			ticker.Stop()
			cancel()
			log.Println("🔌 Cleaning up resources...")
			log.Println("✅ Shutdown complete. Goodbye!")
			return
		}
	}
}

func startup(ctx context.Context) (*config.Config, *monitor.Registry, *notifier.Registry, *time.Ticker) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("❌ Failed to load config: %v", err)
		return nil, nil, nil, nil
	}

	// Initialize monitors
	monitors := monitor.NewRegistry(cfg.ServerName)
	
	cpuThresh := 101.0 // Unreachable by default
	if cfg.CPUThreshold != nil {
		cpuThresh = *cfg.CPUThreshold
		log.Printf("📊 CPU alerts enabled (threshold: %.1f%%)", cpuThresh)
	}
	monitors.Register("cpu", monitor.NewCPUMonitor(cpuThresh))

	memThresh := 101.0
	if cfg.MemoryThreshold != nil {
		memThresh = *cfg.MemoryThreshold
		log.Printf("📊 Memory alerts enabled (threshold: %.1f%%)", memThresh)
	}
	monitors.Register("memory", monitor.NewMemoryMonitor(memThresh))

	diskThresh := 101.0
	if cfg.DiskThreshold != nil {
		diskThresh = *cfg.DiskThreshold
		log.Printf("📊 Disk alerts enabled (threshold: %.1f%%)", diskThresh)
	}
	monitors.Register("disk", monitor.NewDiskMonitor(diskThresh, cfg.ExcludedDirs))
	for _, hc := range cfg.HealthChecks {
		if strings.HasPrefix(hc, "#") {
			continue
		}
		parts := strings.SplitN(hc, ";", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			url := strings.TrimSpace(parts[1])
			monitors.Register("health_"+name, monitor.NewHealthMonitor(name, url))
			log.Printf("📊 Health check enabled for: %s (%s)", name, url)
		}
	}

	// Initialize notifiers
	notifiers := notifier.NewRegistry()
	if cfg.LarkWebhookURL != "" {
		notifiers.Register("lark", notifier.NewLarkNotifier(cfg.LarkWebhookURL))
		log.Println("📢 Lark notifier enabled")
	}

	// Webhook for high-frequency metrics
	if cfg.WebhookURL != "" {
		webhookNotifier := notifier.NewWebhookNotifier(cfg.WebhookURL, cfg.ServerID, cfg.ServerKey)
		log.Printf("🚀 Webhook for metrics enabled (URL: %s, Interval: %s)", cfg.WebhookURL, cfg.WebhookInterval)

		// Start independent metrics loop
		go func(m *monitor.Registry, n *notifier.WebhookNotifier, serverName string) {
			interval, _ := time.ParseDuration(cfg.WebhookInterval)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					metrics := m.CheckMetrics()
					if len(metrics) > 0 {
						n.Notify(metrics)
					}
				case <-ctx.Done():
					return
				}
			}
		}(monitors, webhookNotifier, cfg.ServerName)
	}

	// Database Checks run in their own high-frequency loop
	for _, dbc := range cfg.DBChecks {
		if strings.HasPrefix(dbc, "#") {
			continue
		}
		parts := strings.SplitN(dbc, ";", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			dsn := strings.TrimSpace(parts[1])

			m := monitor.NewDBMonitor(name, dsn)
			log.Printf("📊 Immediate DB monitoring enabled for: %s", name)

			// Start independent check loop
			go func(dbM *monitor.DBMonitor, n *notifier.Registry, serverName string) {
				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						metrics := dbM.CheckMetrics()
						// Add server name to all metrics
						for i := range metrics {
							metrics[i].ServerName = serverName
						}
						alerts := dbM.CheckAlerts(metrics)
						if len(alerts) > 0 {
							log.Printf("⚠️  DB Alert: %s", alerts[0].Message)
							// alerts already have serverName from CheckAlerts
							n.NotifyAll(alerts)
						}
					case <-ctx.Done():
						return
					}
				}
			}(m, notifiers, cfg.ServerName)
		}
	}

	// Parse check interval
	interval, err := time.ParseDuration(cfg.CheckInterval)
	if err != nil {
		log.Printf("❌ Invalid check_interval: %v", err)
		return nil, nil, nil, nil
	}

	log.Printf("✅ Configured to check for alerts every %v", interval)
	return cfg, monitors, notifiers, time.NewTicker(interval)
}

func runAlertChecks(monitors *monitor.Registry, notifiers *notifier.Registry) {
	alerts := monitors.CheckAlerts()

	if len(alerts) > 0 {
		log.Printf("⚠️  Found %d alert(s)", len(alerts))
		for _, alert := range alerts {
			log.Printf("   - %s", alert.Message)
		}
		notifiers.NotifyAll(alerts)
	} else {
		log.Println("✓ All systems nominal for alerts")
	}
}

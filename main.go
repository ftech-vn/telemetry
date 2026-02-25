package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"telemetry/internal/config"
	"telemetry/internal/monitor"
	"telemetry/internal/notifier"
)

var Version = "dev"

func main() {
	log.Println("🚀 Telemetry service starting...")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Initial configuration and startup
	cfg, monitors, notifiers, ticker := startup()
	defer ticker.Stop()

	// Run initial check
	runChecks(monitors, notifiers)

	for {
		select {
		case <-ticker.C:
			runChecks(monitors, notifiers)
		case sig := <-sigChan:
			if sig == syscall.SIGHUP {
				log.Println("♻️  Received SIGHUP, reloading configuration...")
				newCfg, newMonitors, newNotifiers, newTicker := startup()
				if newTicker != nil {
					ticker.Stop()
					cfg, monitors, notifiers, ticker = newCfg, newMonitors, newNotifiers, newTicker
					log.Println("✅ Configuration reloaded successfully")
					// Run check immediately after reload
					runChecks(monitors, notifiers)
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
			log.Println("🔌 Cleaning up resources...")
			log.Println("✅ Shutdown complete. Goodbye!")
			return
		}
	}
}

func startup() (*config.Config, *monitor.Registry, *notifier.Registry, *time.Ticker) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("❌ Failed to load config: %v", err)
		return nil, nil, nil, nil
	}

	// Initialize monitors
	monitors := monitor.NewRegistry(cfg.ServerName)
	if cfg.CPUThreshold != nil {
		monitors.Register("cpu", monitor.NewCPUMonitor(*cfg.CPUThreshold))
		log.Printf("📊 CPU monitoring enabled (threshold: %.1f%%)", *cfg.CPUThreshold)
	}
	if cfg.MemoryThreshold != nil {
		monitors.Register("memory", monitor.NewMemoryMonitor(*cfg.MemoryThreshold))
		log.Printf("📊 Memory monitoring enabled (threshold: %.1f%%)", *cfg.MemoryThreshold)
	}
	if cfg.DiskThreshold != nil {
		monitors.Register("disk", monitor.NewDiskMonitor(*cfg.DiskThreshold, cfg.ExcludedDirs))
		log.Printf("📊 Disk monitoring enabled (threshold: %.1f%%)", *cfg.DiskThreshold)
	}
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
	notifiers.Register("lark", notifier.NewLarkNotifier(cfg.LarkWebhookURL))

	// Parse check interval
	interval, err := time.ParseDuration(cfg.CheckInterval)
	if err != nil {
		log.Printf("❌ Invalid check_interval: %v", err)
		return nil, nil, nil, nil
	}

	log.Printf("✅ Configured to check every %v", interval)
	return cfg, monitors, notifiers, time.NewTicker(interval)
}

func runChecks(monitors *monitor.Registry, notifiers *notifier.Registry) {
	alerts := monitors.CheckAll()
	
	if len(alerts) > 0 {
		log.Printf("⚠️  Found %d alert(s)", len(alerts))
		for _, alert := range alerts {
			log.Printf("   - %s", alert.Message)
		}
		notifiers.NotifyAll(alerts)
	} else {
		log.Println("✓ All systems nominal")
	}
}

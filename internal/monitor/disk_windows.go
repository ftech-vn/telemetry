//go:build windows

package monitor

import (
	"fmt"
	"log"

	"github.com/shirou/gopsutil/v3/disk"
)

type DiskMonitor struct {
	threshold    float64
	excludedDirs []string
}

func NewDiskMonitor(threshold float64, excludedDirs []string) *DiskMonitor {
	return &DiskMonitor{
		threshold:    threshold,
		excludedDirs: excludedDirs,
	}
}

func (m *DiskMonitor) CheckMetrics() []Alert {
	usage, err := disk.Usage("C:")
	if err != nil {
		log.Printf("Error getting usage for C:: %v", err)
		return nil
	}
	return []Alert{
		{
			Type:  "disk",
			Value: usage.UsedPercent,
		},
	}
}

func (m *DiskMonitor) CheckAlerts(metrics []Alert) []Alert {
	var alerts []Alert
	for _, metric := range metrics {
		if metric.Type == "disk" && metric.Value >= m.threshold {
			usage, err := disk.Usage("C:")
			if err != nil {
				log.Printf("Error getting usage for C: for alert details: %v", err)
				continue
			}

			// Dynamic critical threshold
			criticalThreshold := m.threshold + 15.0
			if criticalThreshold > 95.0 {
				criticalThreshold = 95.0
			}
			
			severity := "warning"
			if metric.Value >= criticalThreshold {
				severity = "critical"
			}

			message := fmt.Sprintf("Disk C: is %.1f%% used (%.1f GB / %.1f GB), exceeding threshold of %.1f%%.",
				metric.Value, 
				float64(usage.Used)/1024/1024/1024, 
				float64(usage.Total)/1024/1024/1024, 
				m.threshold)

			alerts = append(alerts, Alert{
				Type:     "disk",
				Message:  message,
				Value:    metric.Value,
				Severity: severity,
			})
		}
	}
	return alerts
}

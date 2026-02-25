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

func (m *DiskMonitor) Check() []Alert {
	usage, err := disk.Usage("C:")
	if err != nil {
		log.Printf("Error getting usage for C:: %v", err)
		return nil
	}

	if usage.UsedPercent < m.threshold {
		return nil
	}

	// Dynamic critical threshold
	criticalThreshold := m.threshold + 15.0
	if criticalThreshold > 95.0 {
		criticalThreshold = 95.0
	}
	
	severity := "warning"
	if usage.UsedPercent >= criticalThreshold {
		severity = "critical"
	}

	message := fmt.Sprintf("Disk C: is %.1f%% used (%.1f GB / %.1f GB), exceeding threshold of %.1f%%.",
		usage.UsedPercent, 
		float64(usage.Used)/1024/1024/1024, 
		float64(usage.Total)/1024/1024/1024, 
		m.threshold)

	return []Alert{
		{
			Type:     "disk",
			Message:  message,
			Value:    usage.UsedPercent,
			Severity: severity,
		},
	}
}

package monitor

import (
	"fmt"
	"log"
	"sort"

	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

type MemoryMonitor struct {
	threshold float64
}

func NewMemoryMonitor(threshold float64) *MemoryMonitor {
	return &MemoryMonitor{
		threshold: threshold,
	}
}

func (m *MemoryMonitor) CheckMetrics() []Alert {
	v, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error getting memory usage: %v", err)
		return nil
	}

	return []Alert{
		{
			Type:  "memory",
			Value: v.UsedPercent,
		},
	}
}

func (m *MemoryMonitor) CheckAlerts(metrics []Alert) []Alert {
	var alerts []Alert
	for _, metric := range metrics {
		if metric.Type == "memory" && metric.Value >= m.threshold {
			v, err := mem.VirtualMemory()
			if err != nil {
				log.Printf("Error getting memory usage for alert details: %v", err)
				continue
			}

			criticalThreshold := m.threshold + 15.0
			if criticalThreshold > 95.0 {
				criticalThreshold = 95.0
			}

			severity := "warning"
			if metric.Value >= criticalThreshold {
				severity = "critical"
			}

			usedMB := float64(v.Used) / 1024 / 1024
			totalMB := float64(v.Total) / 1024 / 1024

			// Get top processes by Memory
			var details string
			procs, err := process.Processes()
			if err == nil {
				type procUsage struct {
					name string
					pid  int32
					rss  uint64
				}
				var usageList []procUsage
				for _, p := range procs {
					mInfo, err := p.MemoryInfo()
					if err == nil && mInfo.RSS > 0 {
						name, _ := p.Name()
						usageList = append(usageList, procUsage{name, p.Pid, mInfo.RSS})
					}
				}
				sort.Slice(usageList, func(i, j int) bool {
					return usageList[i].rss > usageList[j].rss
				})

				details = "\nTop processes by Memory:"
				limit := 5
				if len(usageList) < limit {
					limit = len(usageList)
				}
				for i := 0; i < limit; i++ {
					details += fmt.Sprintf("\n- %s (PID: %d): %.1f MB", usageList[i].name, usageList[i].pid, float64(usageList[i].rss)/1024/1024)
				}
			}

			alerts = append(alerts, Alert{
				Type:     "memory",
				Message:  fmt.Sprintf("Memory usage is %.1f%% (%.1f MB / %.1f MB), exceeding threshold of %.1f%%%s", metric.Value, usedMB, totalMB, m.threshold, details),
				Value:    metric.Value,
				Severity: severity,
			})
		}
	}
	return alerts
}

package monitor

import (
	"fmt"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/process"
)

type CPUMonitor struct {
	threshold float64
}

func NewCPUMonitor(threshold float64) *CPUMonitor {
	return &CPUMonitor{
		threshold: threshold,
	}
}

func (m *CPUMonitor) CheckMetrics() []Alert {
	percents, err := cpu.Percent(time.Second, false)
	if err != nil || len(percents) == 0 {
		return nil
	}
	systemUsage := percents[0]

	return []Alert{
		{
			Type:  "cpu",
			Value: systemUsage,
		},
	}
}

func (m *CPUMonitor) CheckAlerts(metrics []Alert) []Alert {
	var alerts []Alert
	for _, metric := range metrics {
		if metric.Type == "cpu" && metric.Value >= m.threshold {
			// Threshold breached, create a detailed alert
			procs, err := process.Processes()
			if err != nil {
				return nil
			}

			var usageList []struct {
				name    string
				pid     int32
				percent float64
			}

			for _, p := range procs {
				percent, err := p.CPUPercent()
				if err == nil && percent > 0.1 {
					name, _ := p.Name()
					usageList = append(usageList, struct {
						name    string
						pid     int32
						percent float64
					}{name, p.Pid, percent})
				}
			}

			sort.Slice(usageList, func(i, j int) bool {
				return usageList[i].percent > usageList[j].percent
			})

			details := "\nTop processes by CPU:"
			limit := 5
			if len(usageList) < limit {
				limit = len(usageList)
			}
			for i := 0; i < limit; i++ {
				details += fmt.Sprintf("\n- %s (PID: %d): %.1f%%", usageList[i].name, usageList[i].pid, usageList[i].percent)
			}

			criticalThreshold := m.threshold + 15.0
			if criticalThreshold > 95.0 {
				criticalThreshold = 95.0
			}
			severity := "warning"
			if metric.Value >= criticalThreshold {
				severity = "critical"
			}

			alerts = append(alerts, Alert{
				Type:     "cpu",
				Message:  fmt.Sprintf("CPU usage is %.1f%%, exceeding threshold of %.1f%%%s", metric.Value, m.threshold, details),
				Value:    metric.Value,
				Severity: severity,
			})
		}
	}
	return alerts
}

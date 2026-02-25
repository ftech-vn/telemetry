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

func (m *CPUMonitor) Check() []Alert {
	// Get initial system times
	t1, _ := cpu.Times(false)
	// Get initial process times
	procs, _ := process.Processes()
	p1 := make(map[int32]float64)
	pCache := make(map[int32]*process.Process)
	for _, p := range procs {
		if t, err := p.Times(); err == nil {
			p1[p.Pid] = t.User + t.System
			pCache[p.Pid] = p
		}
	}

	// Wait 1 second for sampling
	time.Sleep(time.Second)

	// Get final system times
	t2, _ := cpu.Times(false)
	// Get final process times
	var usageList []struct {
		name    string
		pid     int32
		percent float64
	}

	if len(t1) > 0 && len(t2) > 0 {
		// Total system delta across all cores
		totalDelta := (t2[0].User + t2[0].System + t2[0].Nice + t2[0].Iowait + t2[0].Irq + t2[0].Softirq + t2[0].Steal + t2[0].Idle) -
			(t1[0].User + t1[0].System + t1[0].Nice + t1[0].Iowait + t1[0].Irq + t1[0].Softirq + t1[0].Steal + t1[0].Idle)
		
		// Idle delta
		idleDelta := t2[0].Idle - t1[0].Idle
		systemUsage := (1.0 - idleDelta/totalDelta) * 100

		if systemUsage >= m.threshold {
			for pid, p := range pCache {
				if t, err := p.Times(); err == nil {
					if startTime, ok := p1[pid]; ok {
						procDelta := (t.User + t.System) - startTime
						// Percentage of TOTAL system capacity
						percent := (procDelta / totalDelta) * 100
						if percent > 0.1 {
							name, _ := p.Name()
							usageList = append(usageList, struct {
								name    string
								pid     int32
								percent float64
							}{name, pid, percent})
						}
					}
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
			if systemUsage >= criticalThreshold {
				severity = "critical"
			}

			return []Alert{
				{
					Type:     "cpu",
					Message:  fmt.Sprintf("CPU usage is %.1f%%, exceeding threshold of %.1f%%%s", systemUsage, m.threshold, details),
					Value:    systemUsage,
					Severity: severity,
				},
			}
		}
	}

	return nil
}

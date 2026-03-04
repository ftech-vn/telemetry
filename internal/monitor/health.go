package monitor

import (
	"fmt"
	"net/http"
	"time"
)

type HealthMonitor struct {
	name string
	url  string
}

func NewHealthMonitor(name, url string) *HealthMonitor {
	return &HealthMonitor{
		name: name,
		url:  url,
	}
}

func (m *HealthMonitor) CheckMetrics() []Alert {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(m.url)
	if err != nil {
		// Return a non-200 status code to indicate failure
		return []Alert{
			{
				Type:  "health",
				Value: -1, // Represents a connection error
			},
		}
	}
	defer resp.Body.Close()

	return []Alert{
		{
			Type:  "health",
			Value: float64(resp.StatusCode),
		},
	}
}

func (m *HealthMonitor) CheckAlerts(metrics []Alert) []Alert {
	var alerts []Alert
	for _, metric := range metrics {
		if metric.Type == "health" {
			if metric.Value == -1 {
				alerts = append(alerts, Alert{
					Type:     "health",
					Message:  fmt.Sprintf("🔴 Health check failed for %s (%s): connection error", m.name, m.url),
					Severity: "critical",
				})
			} else if int(metric.Value) != http.StatusOK {
				alerts = append(alerts, Alert{
					Type:     "health",
					Message:  fmt.Sprintf("🔴 Health check for %s (%s) returned status code %d (expected 200)", m.name, m.url, int(metric.Value)),
					Severity: "critical",
				})
			}
		}
	}
	return alerts
}

package monitor

import (
	"sync"
)

type Alert struct {
	ServerID   string  `json:"server_id"`
	ServerName string  `json:"-"` // Ignored by JSON marshaller, used for Lark
	Type       string  `json:"Type"`
	Message    string  `json:"Message"`
	Value      float64 `json:"Value"`
	Severity   string  `json:"Severity"`
}

type Monitor interface {
	CheckMetrics() []Alert
	CheckAlerts(metrics []Alert) []Alert
}

type Registry struct {
	mu         sync.RWMutex
	monitors   map[string]Monitor
	names      []string
	serverID   string
	serverName string
}

func NewRegistry(serverID string, serverName string) *Registry {
	return &Registry{
		monitors:   make(map[string]Monitor),
		names:      []string{},
		serverID:   serverID,
		serverName: serverName,
	}
}

func (r *Registry) Register(name string, m Monitor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.monitors[name] = m
	r.names = append(r.names, name)
}

func (r *Registry) CheckAlerts() []Alert {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allAlerts []Alert
	for _, name := range r.names {
		monitor := r.monitors[name]
		metrics := monitor.CheckMetrics()
		for i := range metrics {
			metrics[i].ServerID = r.serverID
			metrics[i].ServerName = r.serverName
		}

		alerts := monitor.CheckAlerts(metrics)
		for i := range alerts {
			alerts[i].ServerID = r.serverID
			alerts[i].ServerName = r.serverName
		}
		allAlerts = append(allAlerts, alerts...)
	}
	return allAlerts
}

func (r *Registry) CheckMetrics() []Alert {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allMetrics []Alert
	for _, name := range r.names {
		monitor := r.monitors[name]
		metrics := monitor.CheckMetrics()
		for i := range metrics {
			metrics[i].ServerID = r.serverID
			metrics[i].ServerName = r.serverName
		}
		allMetrics = append(allMetrics, metrics...)
	}
	return allMetrics
}

package monitor

import (
	"sync"
)

type Alert struct {
	ServerName string
	Type       string
	Message    string
	Value      float64
	Severity   string
}

type Monitor interface {
	Check() []Alert
}

type Registry struct {
	mu         sync.RWMutex
	monitors   map[string]Monitor
	names      []string
	serverName string
}

func NewRegistry(serverName string) *Registry {
	return &Registry{
		monitors:   make(map[string]Monitor),
		names:      []string{},
		serverName: serverName,
	}
}

func (r *Registry) Register(name string, m Monitor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.monitors[name] = m
	r.names = append(r.names, name)
}

func (r *Registry) CheckAll() []Alert {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var alerts []Alert
	for _, name := range r.names {
		monitor := r.monitors[name]
		monitorAlerts := monitor.Check()
		// Add server name to all alerts
		for i := range monitorAlerts {
			monitorAlerts[i].ServerName = r.serverName
		}
		alerts = append(alerts, monitorAlerts...)
	}
	return alerts
}

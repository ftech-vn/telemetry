package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type DBMonitor struct {
	dbType string
	name   string
	dsn    string
	isDown bool
}

func NewDBMonitor(name, dsn string) *DBMonitor {
	// Auto-detect database type from DSN
	dbType := "mysql"
	cleanDSN := strings.ToLower(dsn)
	if strings.HasPrefix(cleanDSN, "postgres://") || 
	   strings.HasPrefix(cleanDSN, "postgresql://") || 
	   strings.Contains(cleanDSN, "sslmode=") ||
	   strings.Contains(cleanDSN, "host=") {
		dbType = "postgres"
		// Force Postgres timeout if not present
		if !strings.Contains(dsn, "connect_timeout=") {
			separator := "?"
			if strings.Contains(dsn, "?") {
				separator = "&"
			}
			dsn += separator + "connect_timeout=5"
		}
	} else {
		// Force MySQL timeout if not present
		if !strings.Contains(dsn, "timeout=") {
			separator := "?"
			if strings.Contains(dsn, "?") {
				separator = "&"
			}
			dsn += separator + "timeout=5s"
		}
	}

	log.Printf("🔌 Initialized %s monitor for %s", dbType, name)

	return &DBMonitor{
		dbType: dbType,
		name:   name,
		dsn:    dsn,
		isDown: false,
	}
}

func (m *DBMonitor) CheckMetrics() []Alert {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	db, err := sql.Open(m.dbType, m.dsn)
	if err != nil {
		return []Alert{{Type: "database", Value: -1}}
	}
	defer db.Close()

	err = db.PingContext(ctx)
	if err != nil {
		return []Alert{{Type: "database", Value: -1}}
	}

	return []Alert{{Type: "database", Value: 1}}
}

func (m *DBMonitor) CheckAlerts(metrics []Alert) []Alert {
	var alerts []Alert
	for _, metric := range metrics {
		if metric.Type == "database" {
			if metric.Value == -1 { // Failure
				if !m.isDown {
					m.isDown = true
					alerts = append(alerts, Alert{
						Type:     "database",
						Message:  fmt.Sprintf("🔴 Database Connection FAILED: %s", m.name),
						Severity: "critical",
					})
				}
			} else { // Success
				if m.isDown {
					m.isDown = false
					alerts = append(alerts, Alert{
						Type:     "database",
						Message:  fmt.Sprintf("🟢 Database Connection RESTORED: %s", m.name),
						Severity: "warning",
					})
				}
			}
		}
	}
	return alerts
}

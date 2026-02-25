package monitor

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type DBMonitor struct {
	dbType      string
	name        string
	dsn         string
	isDown      bool
}

func NewDBMonitor(name, dsn string) *DBMonitor {
	// Auto-detect database type from DSN
	dbType := "mysql" // Default to mysql
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") || strings.Contains(dsn, "sslmode=") {
		dbType = "postgres"
	}

	return &DBMonitor{
		dbType: dbType,
		name:   name,
		dsn:    dsn,
		isDown: false,
	}
}

func (m *DBMonitor) Check() []Alert {
	db, err := sql.Open(m.dbType, m.dsn)
	if err != nil {
		return m.handleFailure(err)
	}
	defer db.Close()

	// Set a short timeout for the connection
	db.SetConnMaxLifetime(time.Second * 5)
	
	err = db.Ping()
	if err != nil {
		return m.handleFailure(err)
	}

	if m.isDown {
		m.isDown = false
		return []Alert{
			{
				Type:     "database",
				Message:  fmt.Sprintf("🟢 Database Connection RESTORED: %s", m.name),
				Severity: "warning",
			},
		}
	}

	return nil
}

func (m *DBMonitor) handleFailure(err error) []Alert {
	if !m.isDown {
		m.isDown = true
		return []Alert{
			{
				Type:     "database",
				Message:  fmt.Sprintf("🔴 Database Connection FAILED: %s: %v", m.name, err),
				Severity: "critical",
			},
		}
	}
	return nil
}

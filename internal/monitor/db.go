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

func (m *DBMonitor) Check() []Alert {
	// Use a context with timeout for the entire operation
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	log.Printf("🔍 Checking DB [%s] (Dialing %s)...", m.name, m.dbType)
	db, err := sql.Open(m.dbType, m.dsn)
	if err != nil {
		return m.handleFailure(err)
	}
	defer db.Close()

	log.Printf("📡 Pinging DB [%s]...", m.name)
	err = db.PingContext(ctx)
	if err != nil {
		return m.handleFailure(err)
	}

	if m.isDown {
		m.isDown = false
		log.Printf("🟢 DB Nominal: %s", m.name)
		return []Alert{
			{
				Type:     "database",
				Message:  fmt.Sprintf("🟢 Database Connection RESTORED: %s", m.name),
				Severity: "warning",
			},
		}
	}

	log.Printf("✅ DB [%s] is OK", m.name)
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

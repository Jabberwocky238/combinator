package combinator

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// StartHTTPServer starts the Manager HTTP API server
func (m *Manager) StartHTTPServer() error {
	r := gin.Default()

	// Health check
	r.GET("/health", m.healthHandler)

	// Physical RDB management
	r.GET("/api/physical/rdbs", m.listPhysicalRDBsHandler)
	r.GET("/api/physical/rdbs/:id/databases", m.listDatabasesHandler)

	// Combined operation for creating logical RDB
	r.GET("/api/logical/rdb/create", m.createLogicalRDBHandler)

	addr := fmt.Sprintf("%s:%d", m.managerHost, m.managerPort)
	return r.Run(addr)
}

// healthHandler returns health status
func (m *Manager) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// listPhysicalRDBsHandler lists all physical RDBs
func (m *Manager) listPhysicalRDBsHandler(c *gin.Context) {
	ids := m.ListPhysicalRDBs()
	c.JSON(http.StatusOK, gin.H{"rdbs": ids})
}

// listDatabasesHandler lists all databases in a physical RDB
func (m *Manager) listDatabasesHandler(c *gin.Context) {
	id := c.Param("id")
	databases, err := m.ListDatabases(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"databases": databases})
}

// createLogicalRDBHandler creates a logical RDB (db + user + grant)
func (m *Manager) createLogicalRDBHandler(c *gin.Context) {
	physicalRDBID := c.Query("rdb_id")
	dbName := c.Query("db_name")
	username := c.Query("username")
	password := c.Query("password")

	if physicalRDBID == "" || dbName == "" || username == "" || password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rdb_id, db_name, username and password are required"})
		return
	}

	if err := m.CreateLogicalRDB(physicalRDBID, dbName, username, password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logical RDB created successfully"})
}

// CreateDatabase creates a new database using a physical RDB connection
func (m *Manager) CreateDatabase(physicalRDBID, dbName string) error {
	m.mu.RLock()
	physicalRDB, exists := m.physicalRDBs[physicalRDBID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("physical RDB %s not found", physicalRDBID)
	}

	if physicalRDB.Type != "postgres" {
		return fmt.Errorf("create database only supported for postgres")
	}

	// Create database
	query := fmt.Sprintf("CREATE DATABASE %s", dbName)
	_, err := physicalRDB.DB.Exec(query)
	return err
}

// CreateUser creates a new database user
func (m *Manager) CreateUser(physicalRDBID, username, password string) error {
	m.mu.RLock()
	physicalRDB, exists := m.physicalRDBs[physicalRDBID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("physical RDB %s not found", physicalRDBID)
	}

	if physicalRDB.Type != "postgres" {
		return fmt.Errorf("create user only supported for postgres")
	}

	// Create user
	query := fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", username, password)
	_, err := physicalRDB.DB.Exec(query)
	return err
}

// GrantPrivileges grants privileges to a user on a database
func (m *Manager) GrantPrivileges(physicalRDBID, dbName, username string) error {
	m.mu.RLock()
	physicalRDB, exists := m.physicalRDBs[physicalRDBID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("physical RDB %s not found", physicalRDBID)
	}

	if physicalRDB.Type != "postgres" {
		return fmt.Errorf("grant privileges only supported for postgres")
	}

	// Grant all privileges on database
	query := fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s", dbName, username)
	_, err := physicalRDB.DB.Exec(query)
	return err
}

// ListDatabases lists all databases in a physical RDB
func (m *Manager) ListDatabases(physicalRDBID string) ([]string, error) {
	m.mu.RLock()
	physicalRDB, exists := m.physicalRDBs[physicalRDBID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("physical RDB %s not found", physicalRDBID)
	}

	if physicalRDB.Type != "postgres" {
		return nil, fmt.Errorf("list databases only supported for postgres")
	}

	rows, err := physicalRDB.DB.Query("SELECT datname FROM pg_database WHERE datistemplate = false")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return nil, err
		}
		databases = append(databases, dbName)
	}

	return databases, nil
}

// CreateLogicalRDB creates a database, user, and grants privileges in one operation
// This is used to set up a new logical RDB connection for Processor
func (m *Manager) CreateLogicalRDB(physicalRDBID, dbName, username, password string) error {
	// Step 1: Create database
	if err := m.CreateDatabase(physicalRDBID, dbName); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Step 2: Create user
	if err := m.CreateUser(physicalRDBID, username, password); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Step 3: Grant privileges
	if err := m.GrantPrivileges(physicalRDBID, dbName, username); err != nil {
		return fmt.Errorf("failed to grant privileges: %w", err)
	}

	return nil
}

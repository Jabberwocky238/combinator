package manager

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

	addr := fmt.Sprintf("%s:%d", m.host, m.port)
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

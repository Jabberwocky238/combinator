package manager

import (
	"fmt"
)

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

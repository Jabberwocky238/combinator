package combinator

import (
	"database/sql"
	"fmt"
	"sync"

	"jabberwocky238/combinator/core/rdb"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// PhysicalRDB represents a physical database connection with superuser privileges
type PhysicalRDB struct {
	ID   string
	Type string
	DB   *sql.DB
}

// PhysicalKV represents a physical KV store connection
type PhysicalKV struct {
	ID   string
	Type string
	// Connection details will be added when implementing KV
}

// Manager manages physical database connections and provides admin operations
type Manager struct {
	mu sync.RWMutex

	// Physical connections (superuser)
	physicalRDBs map[string]*PhysicalRDB
	physicalKVs  map[string]*PhysicalKV

	// Logical connections (limited user, for application use)
	logicalRDBs map[string]RDB
	logicalKVs  map[string]KV

	// Configuration
	config *Config
	// HTTP server ports
	managerPort int
	managerHost string
	gatewayPort int
	gatewayHost string
}

// NewManager creates a new Manager instance
func NewManager(cfg *Config) *Manager {
	return &Manager{
		physicalRDBs: make(map[string]*PhysicalRDB),
		physicalKVs:  make(map[string]*PhysicalKV),
		logicalRDBs:  make(map[string]RDB),
		logicalKVs:   make(map[string]KV),
		config:       cfg,
		managerPort:  cfg.Manager.Port,
		managerHost:  cfg.Manager.Host,
		gatewayPort:  8899,
		gatewayHost:  "localhost",
	}
}

// Start initializes all connections from config
func (m *Manager) Start() error {
	// Add default memory SQLite with ID "1"
	if err := m.addDefaultSQLite(); err != nil {
		return fmt.Errorf("failed to add default SQLite: %w", err)
	}

	// Load physical RDBs
	for _, rdbCfg := range m.config.PhysicalRDBs {
		if err := m.addPhysicalRDB(rdbCfg); err != nil {
			return fmt.Errorf("failed to add physical RDB %s: %w", rdbCfg.ID, err)
		}
	}

	// Load physical KVs
	for _, kvCfg := range m.config.PhysicalKVs {
		if err := m.addPhysicalKV(kvCfg); err != nil {
			return fmt.Errorf("failed to add physical KV %s: %w", kvCfg.ID, err)
		}
	}

	// Load logical RDBs
	for _, rdbCfg := range m.config.LogicalRDBs {
		if err := m.addLogicalRDB(rdbCfg); err != nil {
			return fmt.Errorf("failed to add logical RDB %s: %w", rdbCfg.ID, err)
		}
	}

	return nil
}

// addDefaultSQLite adds a default memory SQLite with ID "1"
func (m *Manager) addDefaultSQLite() error {
	cfg := LogicalRDBConfig{
		ID:  "1",
		URL: "sqlite://:memory:",
	}
	return m.addLogicalRDB(cfg)
}

// addLogicalRDB adds a logical RDB connection
func (m *Manager) addLogicalRDB(cfg LogicalRDBConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Parse URL
	parsed, err := ParseRDBURL(cfg.URL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	var rdbInstance RDB

	switch parsed.Type {
	case "postgres":
		rdbInstance = rdb.NewPsqlRDB(parsed.Host, parsed.Port, parsed.User, parsed.Password, parsed.DBName)
	case "sqlite":
		rdbInstance = rdb.NewSqliteRDB(parsed.Path)
	default:
		return fmt.Errorf("unsupported RDB type: %s", parsed.Type)
	}

	if err := rdbInstance.Start(); err != nil {
		return err
	}

	m.logicalRDBs[cfg.ID] = rdbInstance
	return nil
}

// GetPhysicalRDB retrieves a physical RDB by ID
func (m *Manager) GetPhysicalRDB(id string) (*PhysicalRDB, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rdb, exists := m.physicalRDBs[id]
	return rdb, exists
}

// ListPhysicalRDBs returns all physical RDB IDs
func (m *Manager) ListPhysicalRDBs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.physicalRDBs))
	for id := range m.physicalRDBs {
		ids = append(ids, id)
	}
	return ids
}

// addPhysicalRDB adds a physical RDB connection
func (m *Manager) addPhysicalRDB(cfg PhysicalRDBConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Parse URL
	parsed, err := ParseRDBURL(cfg.URL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	// Only postgres is supported for physical RDB
	if parsed.Type != "postgres" {
		return fmt.Errorf("physical RDB only supports postgres, got: %s", parsed.Type)
	}

	var db *sql.DB

	// Build connection string from parsed URL
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		parsed.Host, parsed.Port, parsed.User, parsed.Password, parsed.DBName)
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	m.physicalRDBs[cfg.ID] = &PhysicalRDB{
		ID:   cfg.ID,
		Type: parsed.Type,
		DB:   db,
	}

	return nil
}

// addPhysicalKV adds a physical KV connection (placeholder for now)
func (m *Manager) addPhysicalKV(cfg PhysicalKVConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// TODO: Implement KV connection
	m.physicalKVs[cfg.ID] = &PhysicalKV{
		ID:   cfg.ID,
		Type: "redis", // placeholder
	}

	return nil
}

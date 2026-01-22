package manager

import (
	"database/sql"
	"fmt"
	"sync"

	"jabberwocky238/combinator/core/config"
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

	// Configuration
	config *config.Config

	// HTTP server port
	port int
	host string
}

// NewManager creates a new Manager instance
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		physicalRDBs: make(map[string]*PhysicalRDB),
		physicalKVs:  make(map[string]*PhysicalKV),
		config:       cfg,
		port:         cfg.Manager.Port,
		host:         cfg.Manager.Host,
	}
}

// Start initializes all physical connections from config
func (m *Manager) Start() error {
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
func (m *Manager) addPhysicalRDB(cfg config.PhysicalRDBConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var db *sql.DB
	var err error

	switch cfg.Type {
	case "postgres":
		connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)
		db, err = sql.Open("postgres", connStr)
	case "sqlite":
		db, err = sql.Open("sqlite3", cfg.URL)
	default:
		return fmt.Errorf("unsupported RDB type: %s", cfg.Type)
	}

	if err != nil {
		return err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	m.physicalRDBs[cfg.ID] = &PhysicalRDB{
		ID:   cfg.ID,
		Type: cfg.Type,
		DB:   db,
	}

	return nil
}

// addPhysicalKV adds a physical KV connection (placeholder for now)
func (m *Manager) addPhysicalKV(cfg config.PhysicalKVConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// TODO: Implement KV connection
	m.physicalKVs[cfg.ID] = &PhysicalKV{
		ID:   cfg.ID,
		Type: cfg.Type,
	}

	return nil
}

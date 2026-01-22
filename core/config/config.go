package config

// Config represents the entire configuration file
type Config struct {
	Manager      ManagerConfig       `json:"manager"`
	PhysicalRDBs []PhysicalRDBConfig `json:"physical_rdbs"`
	PhysicalKVs  []PhysicalKVConfig  `json:"physical_kvs"`
	LogicalRDBs  []LogicalRDBConfig  `json:"logical_rdbs"`
	LogicalKVs   []LogicalKVConfig   `json:"logical_kvs"`
}

// ManagerConfig defines the manager server settings
type ManagerConfig struct {
	Port int    `json:"port"`
	Host string `json:"host"`
}

// PhysicalRDBConfig defines a physical RDB connection (superuser)
type PhysicalRDBConfig struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // "postgres" or "sqlite"
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	DBName   string `json:"dbname,omitempty"`
	URL      string `json:"url,omitempty"` // for sqlite
}

// PhysicalKVConfig defines a physical KV connection
type PhysicalKVConfig struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // "redis", etc.
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password,omitempty"`
}

// LogicalRDBConfig defines a logical RDB connection (limited user)
type LogicalRDBConfig struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // "postgres" or "sqlite"
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	DBName   string `json:"dbname,omitempty"`
	URL      string `json:"url,omitempty"` // for sqlite
}

// LogicalKVConfig defines a logical KV connection
type LogicalKVConfig struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password,omitempty"`
}

package combinator

type Service interface {
	Start() error
	Type() string
}

type RDB interface {
	Service
	Batch(stmts string) error
	Query(stmt string, args ...any) (data []byte, err error)
	Execute(stmt string, args ...any) (data []byte, err error)
}

type KV interface {
	Service
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
}

type Queue interface {
	Service
}

type S3 interface {
	Service
	Get(key string) ([]byte, error)
	Put(key string, value []byte) error
	Delete(key string) error
	GeneratePresignedUploadURL(key string) (string, error)
	GeneratePresignedDownloadURL(key string) (string, error)
}

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
	ID  string `json:"id"`
	URL string `json:"url"` // postgres://user:pass@host:port/dbname
}

// PhysicalKVConfig defines a physical KV connection
type PhysicalKVConfig struct {
	ID  string `json:"id"`
	URL string `json:"url"` // redis://user:pass@host:port/db
}

// LogicalRDBConfig defines a logical RDB connection (limited user)
type LogicalRDBConfig struct {
	ID  string `json:"id"`
	URL string `json:"url"` // postgres://user:pass@host:port/dbname or sqlite://path/to/db
}

// LogicalKVConfig defines a logical KV connection
type LogicalKVConfig struct {
	ID  string `json:"id"`
	URL string `json:"url"` // redis://user:pass@host:port/db
}

package combinator

type Service interface {
	Start() error
	Close() error
	Type() string
}

type RDB interface {
	Service
	Query(stmt string, args ...any) (data []byte, err error)
	Exec(stmt string, args ...any) error
	Batch(stmt []string, args [][]any) error
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

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

type Server interface {
}

package combinator

type RDB interface {
	Execute(stmts string) (data []byte, err error)
	Start() error
}

type KV interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
}

type Queue interface {
}

type S3 interface {
	Get(key string) ([]byte, error)
	Put(key string, value []byte) error
	Delete(key string) error
	GeneratePresignedUploadURL(key string) (string, error)
	GeneratePresignedDownloadURL(key string) (string, error)
}

type Server interface {
}

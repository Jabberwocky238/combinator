package combinator

type Config struct {
	Rdb []RDBConfig `json:"rdb"`
	Kv  []KVConfig  `json:"kv"`
	S3  []S3Config  `json:"s3"`
}

type RDBConfig struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Metadata any    `json:"metadata,omitempty"`
}

type KVConfig struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Metadata any    `json:"metadata,omitempty"`
}

type S3Config struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Metadata any    `json:"metadata,omitempty"`
}

type DevConfig struct {
	Rdb []string `json:"rdb"`
	Kv  []string `json:"kv"`
	S3  []string `json:"s3"`
}

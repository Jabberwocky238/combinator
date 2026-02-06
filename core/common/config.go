package combinator

type Config struct {
	Metadata MetadataConfig `json:"metadata"`
	Rdb      []RDBConfig    `json:"rdb"`
	Kv       []KVConfig     `json:"kv"`
	S3       []S3Config     `json:"s3"`
}

type MetadataConfig struct {
	UID string `json:"uid"`
}

type RDBConfig struct {
	ID       string `json:"id"`
	Enabled  bool   `json:"enabled,omitempty"`
	URL      string `json:"url"`
	Metadata any    `json:"metadata,omitempty"`
}

type KVConfig struct {
	ID       string `json:"id"`
	Enabled  bool   `json:"enabled,omitempty"`
	URL      string `json:"url"`
	Metadata any    `json:"metadata,omitempty"`
}

type S3Config struct {
	ID       string `json:"id"`
	Enabled  bool   `json:"enabled,omitempty"`
	URL      string `json:"url"`
	Metadata any    `json:"metadata,omitempty"`
}

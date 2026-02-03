package combinator

type Config struct {
	Metadata MetadataConfig `json:"metadata"`
	Rdb      []RDBConfig    `json:"rdb"`
	Kv       []KVConfig     `json:"kv"`
}

type MetadataConfig struct {
	UID string `json:"uid"`
}

type RDBConfig struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled,omitempty"`
	URL     string `json:"url"`
}

type KVConfig struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled,omitempty"`
	URL     string `json:"url"`
}

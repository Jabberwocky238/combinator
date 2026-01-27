package combinator

type Config struct {
	Rdb []RDBConfig `json:"rdb"`
	Kv  []KVConfig  `json:"kv"`
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

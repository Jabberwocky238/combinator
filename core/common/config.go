package combinator

type Config struct {
	Rdb []RDBConfig `json:"rdb"`
}

type RDBConfig struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

package kv

import (
	"context"
	"fmt"
	common "jabberwocky238/combinator/core/common"

	"github.com/redis/go-redis/v9"
)

func init() {
	RegisterKVFactory("redis", func(parsed *ParsedKVURL) (common.KV, error) {
		return NewRedisKV(parsed.Host, parsed.Port, parsed.Password, parsed.DB), nil
	})
}

type RedisKV struct {
	client   *redis.Client
	host     string
	port     int
	password string
	db       int
	ctx      context.Context
}

func NewRedisKV(host string, port int, password string, db int) *RedisKV {
	return &RedisKV{
		host:     host,
		port:     port,
		password: password,
		db:       db,
		ctx:      context.Background(),
	}
}

// Get retrieves a value by key
func (r *RedisKV) Get(key string) ([]byte, error) {
	val, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

// Set stores a value by key
func (r *RedisKV) Set(key string, value []byte) error {
	return r.client.Set(r.ctx, key, value, 0).Err()
}

// Start initializes the Redis connection
func (r *RedisKV) Start() error {
	r.client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", r.host, r.port),
		Password: r.password,
		DB:       r.db,
	})

	// Test connection
	_, err := r.client.Ping(r.ctx).Result()
	return err
}

// Type returns the KV store type
func (r *RedisKV) Type() string {
	return "redis"
}

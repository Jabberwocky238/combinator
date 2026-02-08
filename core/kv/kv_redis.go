//go:build prod
// +build prod

package kv

import (
	"bytes"
	"context"
	"fmt"
	"io"

	common "jabberwocky238/combinator/core/common"
	"jabberwocky238/combinator/core/common/models"

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
func (r *RedisKV) Get(key string, opts *models.KVGetOptions) (io.ReadCloser, error) {
	val, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader([]byte(val))), nil
}

// Set stores a value by key
func (r *RedisKV) Set(key string, value io.Reader, opts *models.KVSetOptions) error {
	// 读取流式数据
	data, err := io.ReadAll(value)
	if err != nil {
		return err
	}

	// 处理 NX/XX 和 TTL
	if opts != nil {
		if opts.NX {
			// SET key value NX EX ttl
			if opts.TTL != nil {
				return r.client.SetNX(r.ctx, key, data, *opts.TTL).Err()
			}
			return r.client.SetNX(r.ctx, key, data, 0).Err()
		}
		if opts.XX {
			// SET key value XX EX ttl
			if opts.TTL != nil {
				return r.client.SetXX(r.ctx, key, data, *opts.TTL).Err()
			}
			return r.client.SetXX(r.ctx, key, data, 0).Err()
		}
		if opts.TTL != nil {
			return r.client.Set(r.ctx, key, data, *opts.TTL).Err()
		}
	}

	return r.client.Set(r.ctx, key, data, 0).Err()
}

// Del deletes a value by key
func (r *RedisKV) Del(key string, opts *models.KVDelOptions) error {
	// CAS 删除：使用 Lua 脚本保证原子性
	if opts != nil && opts.CAS {
		script := `
			if redis.call("get", KEYS[1]) == ARGV[1] then
				return redis.call("del", KEYS[1])
			else
				return 0
			end
		`
		result, err := r.client.Eval(r.ctx, script, []string{key}, opts.Value).Result()
		if err != nil {
			return err
		}
		if result.(int64) == 0 {
			return fmt.Errorf("value mismatch for key: %s", key)
		}
		return nil
	}

	// 普通删除
	return r.client.Del(r.ctx, key).Err()
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

func (r *RedisKV) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// Type returns the KV store type
func (r *RedisKV) Type() string {
	return "redis"
}

package kv

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"

	common "jabberwocky238/combinator/core/common"
	"jabberwocky238/combinator/core/common/models"
)

func init() {
	RegisterKVFactory("memory", func(parsed *ParsedKVURL, log *common.NamespacedLogger) (common.KV, error) {
		return NewMemoryKV(), nil
	})
}

type MemoryKV struct {
	store   map[string][]byte
	expires map[string]time.Time
	mu      sync.RWMutex
}

func NewMemoryKV() *MemoryKV {
	return &MemoryKV{
		store:   make(map[string][]byte),
		expires: make(map[string]time.Time),
	}
}

// Get retrieves a value by key
func (m *MemoryKV) Get(key string, opts *models.KVGetOptions) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 检查是否过期
	if expireTime, exists := m.expires[key]; exists {
		if time.Now().After(expireTime) {
			return nil, fmt.Errorf("key not found: %s", key)
		}
	}

	value, ok := m.store[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// 返回流式接口
	return io.NopCloser(bytes.NewReader(value)), nil
}

// Set stores a value by key
func (m *MemoryKV) Set(key string, value io.Reader, opts *models.KVSetOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查 NX/XX 条件
	_, exists := m.store[key]
	if opts != nil {
		if opts.NX && exists {
			return fmt.Errorf("key already exists: %s", key)
		}
		if opts.XX && !exists {
			return fmt.Errorf("key does not exist: %s", key)
		}
	}

	// 读取流式数据
	data, err := io.ReadAll(value)
	if err != nil {
		return err
	}

	// 存储数据
	m.store[key] = data

	// 设置过期时间
	if opts != nil && opts.TTL != nil {
		m.expires[key] = time.Now().Add(*opts.TTL)
	} else {
		delete(m.expires, key)
	}

	return nil
}

// Del deletes a value by key
func (m *MemoryKV) Del(key string, opts *models.KVDelOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查 key 是否存在
	value, exists := m.store[key]
	if !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	// CAS 删除：检查值是否匹配
	if opts != nil && opts.CAS {
		if !bytes.Equal(value, opts.Value) {
			return fmt.Errorf("value mismatch for key: %s", key)
		}
	}

	// 删除 key
	delete(m.store, key)
	delete(m.expires, key)

	return nil
}

// Start initializes the memory KV store (no-op)
func (m *MemoryKV) Start() error {
	return nil
}

func (m *MemoryKV) Close() error {
	// No resources to clean up for in-memory store
	return nil
}

// Type returns the KV store type
func (m *MemoryKV) Type() string {
	return "memory"
}

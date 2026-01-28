package kv

import (
	"fmt"
	common "jabberwocky238/combinator/core/common"
	"sync"
)

func init() {
	RegisterKVFactory("memory", func(parsed *ParsedKVURL) (common.KV, error) {
		return NewMemoryKV(), nil
	})
}

type MemoryKV struct {
	store map[string][]byte
	mu    sync.RWMutex
}

func NewMemoryKV() *MemoryKV {
	return &MemoryKV{
		store: make(map[string][]byte),
	}
}

// Get retrieves a value by key
func (m *MemoryKV) Get(key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok := m.store[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// Return a copy to prevent external modification
	result := make([]byte, len(value))
	copy(result, value)
	return result, nil
}

// Set stores a value by key
func (m *MemoryKV) Set(key string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store a copy to prevent external modification
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	m.store[key] = valueCopy
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

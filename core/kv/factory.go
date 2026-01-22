package kv

import (
	"fmt"

	common "jabberwocky238/combinator/core/common"
)

// KVFactory is a function that creates a KV instance from a parsed URL
type KVFactory func(*ParsedKVURL) (common.KV, error)

var kvFactories = make(map[string]KVFactory)

// RegisterKVFactory registers a KV factory for a specific type
func RegisterKVFactory(kvType string, factory KVFactory) {
	kvFactories[kvType] = factory
}

// CreateKV creates a KV instance based on the parsed URL
func CreateKV(parsed *ParsedKVURL) (common.KV, error) {
	factory, ok := kvFactories[parsed.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported KV type: %s", parsed.Type)
	}
	return factory(parsed)
}

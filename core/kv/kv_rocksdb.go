//go:build rocksdb
// +build rocksdb

package kv

import (
	"fmt"

	"github.com/linxGnu/grocksdb"
	common "jabberwocky238/combinator/core/common"
)

func init() {
	RegisterKVFactory("rocksdb", func(parsed *ParsedKVURL) (common.KV, error) {
		return NewRocksDBKV(parsed.Path), nil
	})
}

type RocksDBKV struct {
	db   *grocksdb.DB
	path string
	opts *grocksdb.Options
	ro   *grocksdb.ReadOptions
	wo   *grocksdb.WriteOptions
}

func NewRocksDBKV(path string) *RocksDBKV {
	return &RocksDBKV{
		path: path,
	}
}

// Get retrieves a value by key
func (r *RocksDBKV) Get(key string) ([]byte, error) {
	value, err := r.db.GetBytes(r.ro, []byte(key))
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return value, nil
}

// Set stores a value by key
func (r *RocksDBKV) Set(key string, value []byte) error {
	return r.db.Put(r.wo, []byte(key), value)
}

// Start initializes the RocksDB connection
func (r *RocksDBKV) Start() error {
	r.opts = grocksdb.NewDefaultOptions()
	r.opts.SetCreateIfMissing(true)

	db, err := grocksdb.OpenDb(r.opts, r.path)
	if err != nil {
		return err
	}

	r.db = db
	r.ro = grocksdb.NewDefaultReadOptions()
	r.wo = grocksdb.NewDefaultWriteOptions()

	return nil
}

// Type returns the KV store type
func (r *RocksDBKV) Type() string {
	return "rocksdb"
}

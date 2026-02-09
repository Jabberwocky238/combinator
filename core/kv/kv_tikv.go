//go:build prod || kv_tikv
// +build prod kv_tikv

package kv

import (
	"bytes"
	"context"
	"fmt"
	"io"

	common "jabberwocky238/combinator/core/common"
	"jabberwocky238/combinator/core/common/models"

	"github.com/tikv/client-go/v2/txnkv"
)

func init() {
	RegisterKVFactory("tikv", func(parsed *ParsedKVURL) (common.KV, error) {
		return NewTiKV([]string{"tikv-pd-client.tikv.svc.cluster.local:2379"}, parsed.Tenant), nil
	})
}

type TiKV struct {
	client  *txnkv.Client
	pdAddrs []string
	tenant  string
	ctx     context.Context
}

func NewTiKV(pdAddrs []string, tenant string) *TiKV {
	if tenant == "" {
		tenant = "default"
	}
	return &TiKV{
		pdAddrs: pdAddrs,
		tenant:  tenant,
		ctx:     context.Background(),
	}
}

// tenantKey adds tenant prefix to key for isolation
func (t *TiKV) tenantKey(key string) []byte {
	return []byte(fmt.Sprintf("tenant:%s:%s", t.tenant, key))
}

// Get retrieves a value by key
func (t *TiKV) Get(key string, opts *models.KVGetOptions) (io.ReadCloser, error) {
	txn, err := t.client.Begin()
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	val, err := txn.Get(t.ctx, t.tenantKey(key))
	if err != nil {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return io.NopCloser(bytes.NewReader(val)), nil
}

// Set stores a value by key
func (t *TiKV) Set(key string, value io.Reader, opts *models.KVSetOptions) error {
	data, err := io.ReadAll(value)
	if err != nil {
		return err
	}

	txn, err := t.client.Begin()
	if err != nil {
		return err
	}

	tenantKey := t.tenantKey(key)

	// 检查 NX/XX 条件
	if opts != nil {
		_, err := txn.Get(t.ctx, tenantKey)
		exists := err == nil

		if opts.NX && exists {
			txn.Rollback()
			return fmt.Errorf("key already exists: %s", key)
		}
		if opts.XX && !exists {
			txn.Rollback()
			return fmt.Errorf("key does not exist: %s", key)
		}
	}

	if err := txn.Set(tenantKey, data); err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit(t.ctx)
}

// Del deletes a value by key
func (t *TiKV) Del(key string, opts *models.KVDelOptions) error {
	txn, err := t.client.Begin()
	if err != nil {
		return err
	}

	tenantKey := t.tenantKey(key)

	// CAS 删除：检查值是否匹配
	if opts != nil && opts.CAS {
		val, err := txn.Get(t.ctx, tenantKey)
		if err != nil {
			txn.Rollback()
			return fmt.Errorf("key not found: %s", key)
		}

		if !bytes.Equal(val, opts.Value) {
			txn.Rollback()
			return fmt.Errorf("value mismatch for key: %s", key)
		}
	}

	if err := txn.Delete(tenantKey); err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit(t.ctx)
}

// Start initializes the TiKV connection
func (t *TiKV) Start() error {
	client, err := txnkv.NewClient(t.pdAddrs)
	if err != nil {
		return fmt.Errorf("failed to connect to TiKV: %w", err)
	}
	t.client = client
	return nil
}

func (t *TiKV) Close() error {
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}

func (t *TiKV) Type() string {
	return "tikv"
}

package models

import "time"

// KVGetOptions 获取选项
type KVGetOptions struct {
	// 预留字段，暂时为空
}

// KVSetOptions 设置选项
type KVSetOptions struct {
	// TTL: 过期时间
	TTL *time.Duration `json:"ttl,omitempty"`
	// NX: 仅当 key 不存在时设置
	NX bool `json:"nx,omitempty"`
	// XX: 仅当 key 存在时设置
	XX bool `json:"xx,omitempty"`
}

// KVDelOptions 删除选项
type KVDelOptions struct {
	// CAS: 是否启用 CAS 删除
	CAS bool `json:"cas,omitempty"`
	// Value: CAS 删除时需要匹配的值（从请求体读取）
	Value []byte `json:"-"`
}

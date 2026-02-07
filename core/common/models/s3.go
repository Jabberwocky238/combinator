package models

import (
	"time"
)

// S3ObjectInfo 对象元数据信息
type S3ObjectInfo struct {
	Key          string            // 对象键名
	Size         int64             // 对象大小（字节）
	LastModified time.Time         // 最后修改时间
	ETag         string            // ETag标识
	ContentType  string            // MIME类型
	Metadata     map[string]string // 自定义元数据
}

// S3ListOptions 列表查询选项
type S3ListOptions struct {
	Prefix     string // 前缀过滤
	MaxKeys    int    // 最大返回数量
	StartAfter string // 分页游标（从此key之后开始）
	Delimiter  string // 目录分隔符（通常为"/"）
}

// S3ListResult 列表查询结果
type S3ListResult struct {
	Objects     []S3ObjectInfo // 对象列表
	Prefixes    []string       // 公共前缀（目录）
	IsTruncated bool           // 是否还有更多结果
	NextMarker  string         // 下一页的游标
}

// S3PutOptions 上传选项
type S3PutOptions struct {
	ContentType string            // MIME类型
	Metadata    map[string]string // 自定义元数据
	Overwrite   bool              // 是否覆盖已存在对象（默认true）
}

// S3GetOptions 下载选项
type S3GetOptions struct {
	Range *S3Range // 范围下载
}

// S3Range 字节范围
type S3Range struct {
	Start int64 // 起始位置
	End   int64 // 结束位置
}

// S3DeleteMode 删除模式
type S3DeleteMode string

const (
	S3DeleteModePrecise S3DeleteMode = "precise" // 精确匹配
	S3DeleteModePrefix  S3DeleteMode = "prefix"  // 前缀匹配
)

// S3DeleteKey 删除键配置
type S3DeleteKey struct {
	Mode S3DeleteMode `json:"mode"` // 删除模式
	Key  string       `json:"key"`  // 键名或前缀
}

// S3DeleteOptions 删除选项
type S3DeleteOptions struct {
	Keys []S3DeleteKey `json:"keys"` // 要删除的键列表
}

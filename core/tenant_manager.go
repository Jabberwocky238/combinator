package combinator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
	kvModule "jabberwocky238/combinator/core/kv"
	rdbModule "jabberwocky238/combinator/core/rdb"
	s3Module "jabberwocky238/combinator/core/s3"
)

var (
	ControlPlaneURL = "http://control-plane.console.svc.cluster.local:9901"
	CockroachDBHost = "cockroachdb-public.cockroachdb.svc.cluster.local"
	CockroachDBPort = "26257"
)

// TenantResource 租户资源信息
type TenantResource struct {
	ResourceID   string `json:"resource_id"`
	ResourceType string `json:"resource_type"` // "rdb", "kv", "s3"
}

func TenantResourceID(userUid, resourceID string) string {
	return fmt.Sprintf("%s-%s", userUid, resourceID)
}

// TenantInfo 租户信息
type TenantInfo struct {
	UID       string           `json:"uid"`
	SecretKey string           `json:"secret_key"`
	Resources []TenantResource `json:"resources"`
	CreatedAt time.Time        `json:"created_at"`
}

// MultiTenantManager 多租户管理器
type MultiTenantManager struct {
	ctx        context.Context
	mu         sync.RWMutex
	tenants    map[string]*TenantInfo // UID -> TenantInfo
	rdbGateway *rdbModule.RDBGateway
	kvGateway  *kvModule.KVGateway
	s3Gateway  *s3Module.S3Gateway
	httpClient *http.Client
}

// NewMultiTenantManager 创建多租户管理器
func NewMultiTenantManager(
	ctx context.Context,
	rdbGateway *rdbModule.RDBGateway,
	kvGateway *kvModule.KVGateway,
	s3Gateway *s3Module.S3Gateway,
) *MultiTenantManager {
	t := MultiTenantManager{
		ctx:        ctx,
		tenants:    make(map[string]*TenantInfo),
		rdbGateway: rdbGateway,
		kvGateway:  kvGateway,
		s3Gateway:  s3Gateway,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
	rdbGateway.TenantHandler = t.EnsureResourceExistsMiddleware("rdb", "rdb_id")
	kvGateway.TenantHandler = t.EnsureResourceExistsMiddleware("kv", "kv_id")
	s3Gateway.TenantHandler = t.EnsureResourceExistsMiddleware("s3", "s3_id")
	return &t
}

// GetTenant 获取租户信息
func (m *MultiTenantManager) GetTenant(uid string) (*TenantInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenant, ok := m.tenants[uid]
	return tenant, ok
}

// SetTenant 设置租户信息
func (m *MultiTenantManager) SetTenant(uid string, tenant *TenantInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenants[uid] = tenant
}

// QueryBackendResponse 后台查询响应
type QueryBackendResponse struct {
	SecretKey string           `json:"secret_key"`
	Resources []TenantResource `json:"resources"`
}

// QueryBackend 查询后台获取租户信息
func (m *MultiTenantManager) QueryBackend(uid string) (*TenantInfo, error) {
	url := fmt.Sprintf("%s/combinator/retrieveSecretByID?user_id=%s", ControlPlaneURL, uid)

	resp, err := m.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query backend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	var result QueryBackendResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	tenant := &TenantInfo{
		UID:       uid,
		SecretKey: result.SecretKey,
		Resources: result.Resources,
		CreatedAt: time.Now(),
	}

	return tenant, nil
}

// EnsureResourceExists 确保资源存在，如果不存在则创建
func (m *MultiTenantManager) EnsureResourceExists(uid, resourceType, resourceID string) error {
	tnid := TenantResourceID(uid, resourceID)
	switch resourceType {
	case "rdb":
		if _, ok := m.rdbGateway.Get(tnid); !ok {
			dbname := fmt.Sprintf("db_%s", uid)
			username := fmt.Sprintf("user_%s", uid)
			url := fmt.Sprintf("postgresql://%s@%s:%s/%s?sslmode=disable&search_path=%s",
				username, CockroachDBHost, CockroachDBPort, dbname, resourceID)
			resource, err := m.rdbGateway.ParseAndCreate(common.RDBConfig{ID: resourceID, URL: url})
			if err != nil {
				return fmt.Errorf("failed to create RDB resource %s: %w", resourceID, err)
			}
			if err := m.rdbGateway.Set(tnid, resource); err != nil {
				return fmt.Errorf("failed to set RDB resource %s: %w", resourceID, err)
			}
			return fmt.Errorf("rdb resource %s does not exist and needs to be created", resourceID)
		}
	case "kv":
		if _, ok := m.kvGateway.Get(tnid); !ok {
			resource, err := m.kvGateway.ParseAndCreate(common.KVConfig{ID: resourceID})
			if err != nil {
				return fmt.Errorf("failed to create KV resource %s: %w", resourceID, err)
			}
			if err := m.kvGateway.Set(tnid, resource); err != nil {
				return fmt.Errorf("failed to set KV resource %s: %w", resourceID, err)
			}
			return fmt.Errorf("kv resource %s does not exist and needs to be created", resourceID)
		}
	case "s3":
		if _, ok := m.s3Gateway.Get(tnid); !ok {
			return fmt.Errorf("s3 resource %s does not exist and needs to be created", resourceID)
		}
	default:
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return nil
}

// DeleteTenantResource 删除租户的指定资源
func (m *MultiTenantManager) DeleteTenantResource(uid, resourceType, resourceID string) error {
	// 从租户信息中移除资源
	m.mu.Lock()
	tenant, ok := m.tenants[uid]
	if ok {
		newResources := make([]TenantResource, 0)
		for _, res := range tenant.Resources {
			if !(res.ResourceType == resourceType && res.ResourceID == resourceID) {
				newResources = append(newResources, res)
			}
		}
		tenant.Resources = newResources
	}
	m.mu.Unlock()
	tnid := TenantResourceID(uid, resourceID)

	// 从对应的 gateway 中删除资源
	switch resourceType {
	case "rdb":
		return m.rdbGateway.Del(tnid)
	case "kv":
		return m.kvGateway.Del(tnid)
	case "s3":
		return m.s3Gateway.Del(tnid)
	default:
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

// Middleware 多租户中间件（包含签名验证）
func (m *MultiTenantManager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		signature := c.GetHeader("X-Raysail-Signature")
		uid := c.GetHeader("X-Raysail-UID")
		c.Set("tenant_mode", false)
		// 如果没有提供 UID，跳过多租户处理
		if uid == "" {
			c.Next()
			return
		}

		// 必须同时提供 signature 和 uid
		if signature == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Missing X-Raysail-Signature header",
			})
			c.Abort()
			return
		}

		// 检查租户是否存在
		tenant, ok := m.GetTenant(uid)
		if !ok {
			// 租户不存在，查询后台
			var err error
			tenant, err = m.QueryBackend(uid)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": fmt.Sprintf("Failed to authenticate tenant: %v", err),
				})
				c.Abort()
				return
			}

			// 保存租户信息
			m.SetTenant(uid, tenant)
		}

		// 验证 HMAC 签名
		body, err := c.GetRawData()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Failed to read request body",
			})
			c.Abort()
			return
		}

		if err := common.VerifyHMACSignature(tenant.SecretKey, body, signature); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid signature",
			})
			c.Abort()
			return
		}

		// 将 body 重新写回请求中，以便后续处理
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// 将租户信息存入上下文
		c.Set("tenant_uid", uid)
		c.Set("tenant_info", tenant)
		c.Set("tenant_mode", true)
		c.Next()
	}
}

// EnsureRDBExistsMiddleware 确保 RDB 资源存在的中间件
func (m *MultiTenantManager) EnsureResourceExistsMiddleware(resourceType string, pickerStr string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 rdb_id（由第一个中间件设置）
		resourceID := c.GetString(pickerStr)
		tenant, exists := c.Get("tenant_info")
		if resourceID == "" || !exists {
			c.Next()
			return
		}
		tenantInfo := tenant.(*TenantInfo)

		if err := m.EnsureResourceExists(tenantInfo.UID, resourceType, resourceID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to ensure %s resource exists: %v", resourceType, err),
			})
			c.Abort()
			return
		}
		c.Set(pickerStr, TenantResourceID(tenantInfo.UID, resourceID))
		c.Next()
	}
}

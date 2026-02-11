package common

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"maps"
	"sync"
)

// ConfigWithID 配置必须提供 ID
type ConfigWithID interface {
	GetID() string
}

// Gateway 泛型网关接口
type Gateway[ServiceTy Service, ConfTy ConfigWithID] interface {
	Service // 继承 Start/Close/Type

	// 解析配置并创建服务实例
	ParseAndCreate(conf ConfTy) (ServiceTy, error)

	// 管理服务实例
	Set(id string, service ServiceTy) error
	Get(id string) (ServiceTy, bool)
	Del(id string) error

	// 批量修改配置
	ModifyConfig(incre []ConfTy, decre []string) error
}

// BaseGateway 提供通用的网关实现
type BaseGateway[ServiceTy Service, ConfTy ConfigWithID] struct {
	mu         sync.RWMutex
	serviceMap map[string]ServiceTy
	InitConf   []ConfTy
	parser     func(ConfTy) (ServiceTy, error)
}

// NewBaseGateway 创建基础网关
func NewBaseGateway[ServiceTy Service, ConfTy ConfigWithID](
	initConf []ConfTy,
	parser func(ConfTy) (ServiceTy, error),
) *BaseGateway[ServiceTy, ConfTy] {
	return &BaseGateway[ServiceTy, ConfTy]{
		serviceMap: make(map[string]ServiceTy),
		InitConf:   initConf,
		parser:     parser,
	}
}

// ParseAndCreate 解析配置并创建服务实例
func (bg *BaseGateway[ServiceTy, ConfTy]) ParseCreateRun(conf ConfTy) (ServiceTy, error) {
	service, err := bg.parser(conf)
	if err != nil {
		return service, err
	}
	if err := service.Start(); err != nil {
		return service, err
	}
	return service, nil
}

// Set 设置服务实例
func (bg *BaseGateway[ServiceTy, ConfTy]) Set(id string, service ServiceTy) error {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	bg.serviceMap[id] = service
	return nil
}

// Get 获取服务实例
func (bg *BaseGateway[ServiceTy, ConfTy]) Get(id string) (ServiceTy, bool) {
	bg.mu.RLock()
	defer bg.mu.RUnlock()
	service, ok := bg.serviceMap[id]
	return service, ok
}

// Del 删除服务实例
func (bg *BaseGateway[ServiceTy, ConfTy]) Del(id string) error {
	bg.mu.Lock()
	defer bg.mu.Unlock()

	if service, ok := bg.serviceMap[id]; ok {
		service.Close()
		delete(bg.serviceMap, id)
	}
	return nil
}

// ModifyConfig 批量修改配置
func (bg *BaseGateway[ServiceTy, ConfTy]) ModifyConfig(incre []ConfTy, decre []string) error {
	bg.mu.Lock()
	defer bg.mu.Unlock()

	// 删除服务
	for _, id := range decre {
		if service, ok := bg.serviceMap[id]; ok {
			service.Close()
			delete(bg.serviceMap, id)
			Logger.Infof("Deleted service: %s", id)
		}
	}

	// 添加新服务
	for _, conf := range incre {
		// 如果id存在则跳过
		id := conf.GetID()
		if _, exists := bg.serviceMap[id]; exists {
			Logger.Infof("Service %s already exists, skipping", id)
			continue
		}
		service, err := bg.parser(conf)
		if err != nil {
			Logger.Errorf("Failed to create service %s: %v", id, err)
			continue
		}

		if err := service.Start(); err != nil {
			Logger.Errorf("Failed to start service %s: %v", id, err)
			continue
		}

		bg.serviceMap[id] = service
		Logger.Infof("Added service: %s (%s)", id, service.Type())
	}

	return nil
}

// GetAllServices 获取所有服务实例
func (bg *BaseGateway[ServiceTy, ConfTy]) GetAllServices() map[string]ServiceTy {
	bg.mu.RLock()
	defer bg.mu.RUnlock()

	result := make(map[string]ServiceTy, len(bg.serviceMap))
	maps.Copy(result, bg.serviceMap)
	return result
}

// GenerateHMACSignature generates HMAC-SHA256 signature
func GenerateHMACSignature(secretKey string, data []byte) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write(data)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// VerifyHMACSignature verifies HMAC-SHA256 signature
func VerifyHMACSignature(secretKey string, data []byte, signature string) error {
	expected := GenerateHMACSignature(secretKey, data)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return errors.New("invalid signature")
	}
	return nil
}

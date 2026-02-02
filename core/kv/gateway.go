package kv

import (
	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
)

type KVGateway struct {
	grg    *gin.RouterGroup
	KvConf []common.KVConfig
	KvMap  map[string]common.KV
}

func NewGateway(grg *gin.RouterGroup, conf []common.KVConfig) *KVGateway {
	return &KVGateway{
		grg:    grg,
		KvConf: conf,
		KvMap:  make(map[string]common.KV),
	}
}

func (gw *KVGateway) loadKVs() error {
	for _, kvConf := range gw.KvConf {
		parsed, err := ParseKVURL(kvConf.URL)
		if err != nil {
			common.Logger.Errorf("Failed to parse KV URL for %s: %v", kvConf.ID, err)
			return err
		}

		// Use factory to create KV instance
		kv, err := CreateKV(parsed)
		if err != nil {
			common.Logger.Errorf("Failed to create KV %s: %v", kvConf.ID, err)
			return err
		}

		gw.KvMap[kvConf.ID] = kv

		if err = kv.Start(); err != nil {
			common.Logger.Errorf("Failed to start KV %s: %v", kvConf.ID, err)
			return err
		}
		common.Logger.Infof("Loaded %s KV: %s", parsed.Type, kvConf.ID)
	}
	return nil
}

func (gw *KVGateway) Start() error {
	err := gw.Reload(gw.KvConf)
	if err != nil {
		return err
	}

	// KV 路由组
	gw.grg.Use(gw.middlewareKV())
	{
		gw.grg.GET("/get", gw.handleGet)
		gw.grg.POST("/set", gw.handleSet)
	}

	return nil
}

func (gw *KVGateway) middlewareKV() gin.HandlerFunc {
	return func(c *gin.Context) {
		kvID := c.GetHeader("X-Combinator-KV-ID")
		if kvID == "" {
			c.JSON(400, gin.H{"error": "missing X-Combinator-KV-ID header"})
			c.Abort()
			return
		}

		key := c.GetHeader("X-Combinator-KV-Key")
		if key == "" {
			c.JSON(400, gin.H{"error": "missing X-Combinator-KV-Key header"})
			c.Abort()
			return
		}

		// 注入 KV ID 和 Key 到 context
		c.Set("kv_id", kvID)
		c.Set("kv_key", key)
		c.Next()
	}
}

func (gw *KVGateway) handleGet(c *gin.Context) {
	kv := gw.KvMap[c.GetString("kv_id")]
	if kv == nil {
		c.JSON(400, gin.H{"error": "invalid KV ID"})
		return
	}

	key := c.GetString("kv_key")

	value, err := kv.Get(key)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.Data(200, "application/octet-stream", value)
}

func (gw *KVGateway) handleSet(c *gin.Context) {
	kv := gw.KvMap[c.GetString("kv_id")]
	if kv == nil {
		c.JSON(400, gin.H{"error": "invalid KV ID"})
		return
	}

	key := c.GetString("kv_key")

	value, err := c.GetRawData()
	if err != nil {
		c.JSON(400, gin.H{"error": "failed to read request body"})
		return
	}

	if err := kv.Set(key, value); err != nil {
		common.Logger.Errorf("Set failed: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.String(200, "OK")
}

// Reload 重新加载 KV 配置
func (gw *KVGateway) Reload(newConf []common.KVConfig) error {
	// 构建新配置的 ID 集合
	newIDs := make(map[string]common.KVConfig)
	for _, conf := range newConf {
		newIDs[conf.ID] = conf
	}

	// 创建新的 KV map
	newKVMap := make(map[string]common.KV)

	// 1. 保留未变化的 KV
	for id, kv := range gw.KvMap {
		if newConf, exists := newIDs[id]; exists {
			// 检查配置是否变化
			oldConf := gw.findConfigByID(id)
			if oldConf != nil && oldConf.URL == newConf.URL {
				// 配置未变化，保留
				newKVMap[id] = kv
				common.Logger.Infof("KV %s unchanged, keeping connection", id)
				delete(newIDs, id)
				continue
			}
		}
		// 配置变化或被删除，关闭旧连接
		if err := kv.Close(); err != nil {
			common.Logger.Warnf("Failed to close KV %s: %v", id, err)
		}
		common.Logger.Infof("Closed KV %s", id)
	}

	// 2. 加载新增或变化的 KV
	for id, conf := range newIDs {
		parsed, err := ParseKVURL(conf.URL)
		if err != nil {
			common.Logger.Errorf("Failed to parse KV URL for %s: %v", id, err)
			return err
		}

		kv, err := CreateKV(parsed)
		if err != nil {
			common.Logger.Errorf("Failed to create KV %s: %v", id, err)
			return err
		}

		if err = kv.Start(); err != nil {
			common.Logger.Errorf("Failed to start KV %s: %v", id, err)
			return err
		}

		newKVMap[id] = kv
		common.Logger.Infof("Loaded %s KV: %s", parsed.Type, id)
	}

	// 3. 更新配置和 map
	gw.KvMap = newKVMap
	gw.KvConf = newConf

	return nil
}

// findConfigByID 查找配置
func (gw *KVGateway) findConfigByID(id string) *common.KVConfig {
	for _, conf := range gw.KvConf {
		if conf.ID == id {
			return &conf
		}
	}
	return nil
}

package kv

import (
	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
)

type KVGateway struct {
	grg    *gin.RouterGroup
	kvConf []common.KVConfig
	kvMap  map[string]common.KV
}

func NewGateway(grg *gin.RouterGroup, conf []common.KVConfig) *KVGateway {
	return &KVGateway{
		grg:    grg,
		kvConf: conf,
		kvMap:  make(map[string]common.KV),
	}
}

func (gw *KVGateway) loadKVs() error {
	for _, kvConf := range gw.kvConf {
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

		gw.kvMap[kvConf.ID] = kv

		if err = kv.Start(); err != nil {
			common.Logger.Errorf("Failed to start KV %s: %v", kvConf.ID, err)
			return err
		}
		common.Logger.Infof("Loaded %s KV: %s", parsed.Type, kvConf.ID)
	}
	return nil
}

func (gw *KVGateway) Start() error {
	err := gw.loadKVs()
	if err != nil {
		return err
	}

	// KV 路由组
	gw.grg.Use(middlewareKV())
	{
		gw.grg.GET("/get", gw.handleGet)
		gw.grg.POST("/set", gw.handleSet)
	}

	return nil
}

func middlewareKV() gin.HandlerFunc {
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
	kv := gw.kvMap[c.GetString("kv_id")]
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
	kv := gw.kvMap[c.GetString("kv_id")]
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

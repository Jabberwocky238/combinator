package kv

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
	"jabberwocky238/combinator/core/common/models"
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
	// KV 路由组
	gw.grg.Use(gw.middlewareKV())
	{
		gw.grg.GET("/get", gw.handleGet)
		gw.grg.POST("/set", gw.handleSet)
		gw.grg.POST("/del", gw.handleDelete)
	}

	return gw.Reload(gw.KvConf)
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

		// 获取 KV 实例
		kv := gw.KvMap[kvID]
		if kv == nil {
			c.JSON(400, gin.H{"error": "invalid KV ID"})
			c.Abort()
			return
		}

		// 获取 pathname
		pathname := c.Request.URL.Path

		// 解析 options (JSON 格式)，根据 pathname 决定解析类型
		optionsStr := c.GetHeader("X-Combinator-KV-Options")
		if optionsStr != "" {
			switch {
			case strings.HasSuffix(pathname, "/set"):
				var setOpts models.KVSetOptions
				if err := json.Unmarshal([]byte(optionsStr), &setOpts); err == nil {
					c.Set("kv_set_options", &setOpts)
				}
			case strings.HasSuffix(pathname, "/del"):
				var delOpts models.KVDelOptions
				if err := json.Unmarshal([]byte(optionsStr), &delOpts); err == nil {
					c.Set("kv_del_options", &delOpts)
					// 如果启用 CAS，标记需要读取请求体
					c.Set("kv_del_cas", delOpts.CAS)
				}
			}
		}

		// 注入 KV 实例和 Key 到 context
		c.Set("kv", kv)
		c.Set("kv_key", key)
		c.Next()
	}
}

func (gw *KVGateway) handleGet(c *gin.Context) {
	kv := c.MustGet("kv").(common.KV)
	key := c.GetString("kv_key")

	reader, err := kv.Get(key, nil)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()

	c.Stream(func(w io.Writer) bool {
		io.Copy(w, reader)
		return false
	})
}

func (gw *KVGateway) handleSet(c *gin.Context) {
	kv := c.MustGet("kv").(common.KV)
	key := c.GetString("kv_key")

	// 从 context 获取解析好的 options
	var opts *models.KVSetOptions
	if val, exists := c.Get("kv_set_options"); exists {
		opts = val.(*models.KVSetOptions)
	}

	// 使用流式接口
	if err := kv.Set(key, c.Request.Body, opts); err != nil {
		common.Logger.Errorf("Set failed: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.String(200, "OK")
}

func (gw *KVGateway) handleDelete(c *gin.Context) {
	kv := c.MustGet("kv").(common.KV)
	key := c.GetString("kv_key")

	// 从 context 获取解析好的 options
	var opts *models.KVDelOptions
	if val, exists := c.Get("kv_del_options"); exists {
		opts = val.(*models.KVDelOptions)
	}

	// 如果启用 CAS，从请求体读取 Value
	if opts != nil && opts.CAS {
		value, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(400, gin.H{"error": "failed to read request body"})
			return
		}
		opts.Value = value
	}

	if err := kv.Del(key, opts); err != nil {
		common.Logger.Errorf("Delete failed: %v", err)
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

// parseKVSetOptions 解析 Set 选项
// 格式: ttl=10s&nx=true&xx=false
func parseKVSetOptions(optionsStr string) *models.KVSetOptions {
	if optionsStr == "" {
		return nil
	}

	opts := &models.KVSetOptions{}
	params := strings.Split(optionsStr, "&")

	for _, param := range params {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "ttl":
			if duration, err := time.ParseDuration(value); err == nil {
				opts.TTL = &duration
			}
		case "nx":
			opts.NX = value == "true"
		case "xx":
			opts.XX = value == "true"
		}
	}

	return opts
}

// parseKVDelOptions 解析 Delete 选项
// 格式: value=token123
func parseKVDelOptions(optionsStr string) *models.KVDelOptions {
	if optionsStr == "" {
		return nil
	}

	opts := &models.KVDelOptions{}
	params := strings.Split(optionsStr, "&")

	for _, param := range params {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		if key == "value" {
			opts.Value = []byte(value)
		}
	}

	return opts
}

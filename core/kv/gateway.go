package kv

import (
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
	"jabberwocky238/combinator/core/common/models"
)

type KVGateway struct {
	*common.BaseGateway[common.KV, common.KVConfig]
	grg           *gin.RouterGroup
	TenantHandler gin.HandlerFunc
}

func NewGateway(grg *gin.RouterGroup, conf []common.KVConfig) *KVGateway {
	parser := func(c common.KVConfig) (common.KV, error) {
		parsed, err := ParseKVURL(c.URL)
		if err != nil {
			return nil, err
		}
		return CreateKV(parsed)
	}

	return &KVGateway{
		BaseGateway: common.NewBaseGateway(conf, parser),
		grg:         grg,
	}
}

func (gw *KVGateway) Start() error {
	// 使用 ModifyConfig 加载初始配置
	if err := gw.ModifyConfig(gw.InitConf, nil); err != nil {
		return err
	}

	// KV 路由组
	gw.grg.Use(gw.middlewareKV())
	if gw.TenantHandler != nil {
		gw.grg.Use(gw.TenantHandler)
	}
	gw.grg.Use(gw.middlewareCatchKV())
	{
		gw.grg.GET("/get", gw.handleGet)
		gw.grg.POST("/set", gw.handleSet)
		gw.grg.POST("/del", gw.handleDelete)
	}

	return nil
}

func (gw *KVGateway) Close() error {
	for _, kv := range gw.GetAllServices() {
		kv.Close()
	}
	return nil
}

func (gw *KVGateway) Type() string {
	return "kv-gateway"
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

		// 获取 pathname
		pathname := c.Request.URL.Path

		// 解析 options (JSON 格式)，根据 pathname 决定解析类型
		optionsStr := c.GetHeader("X-Combinator-KV-Options")
		if optionsStr != "" {
			switch {
			case strings.HasSuffix(pathname, "/set"):
				setOpts := parseKVSetOptions(optionsStr)
				if setOpts != nil {
					c.Set("kv_set_options", setOpts)
				}
			case strings.HasSuffix(pathname, "/del"):
				delOpts := parseKVDelOptions(optionsStr)
				if delOpts != nil {
					c.Set("kv_del_options", delOpts)
				}
			}
		}

		c.Set("kv_key", key)
		c.Next()
	}
}

func (gw *KVGateway) middlewareCatchKV() gin.HandlerFunc {
	return func(c *gin.Context) {
		kvID := c.MustGet("kv_id").(string)
		// 获取 KV 实例
		kv, ok := gw.Get(kvID)
		if !ok {
			c.JSON(400, gin.H{"error": "invalid KV ID"})
			c.Abort()
			return
		}
		c.Set("kv", kv)
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

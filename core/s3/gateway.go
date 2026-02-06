package s3

import (
	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
)

type S3Gateway struct {
	grg    *gin.RouterGroup
	S3Conf []common.S3Config
	S3Map  map[string]common.S3
}

func NewGateway(grg *gin.RouterGroup, conf []common.S3Config) *S3Gateway {
	return &S3Gateway{
		grg:    grg,
		S3Conf: conf,
		S3Map:  make(map[string]common.S3),
	}
}

func (gw *S3Gateway) Start() error {
	err := gw.Reload(gw.S3Conf)
	if err != nil {
		return err
	}

	gw.grg.Use(gw.middlewareS3())
	{
		gw.grg.GET("/get", gw.handleGet)
		gw.grg.POST("/put", gw.handlePut)
		gw.grg.GET("/list", gw.handleList)
		gw.grg.DELETE("/delete", gw.handleDelete)
	}

	return nil
}

func (gw *S3Gateway) middlewareS3() gin.HandlerFunc {
	return func(c *gin.Context) {
		s3ID := c.GetHeader("X-Combinator-S3-ID")
		if s3ID == "" {
			c.JSON(400, gin.H{"error": "missing X-Combinator-S3-ID header"})
			c.Abort()
			return
		}
		c.Set("s3_id", s3ID)
		c.Next()
	}
}

func (gw *S3Gateway) handleGet(c *gin.Context) {
	s3 := gw.S3Map[c.GetString("s3_id")]
	if s3 == nil {
		c.JSON(400, gin.H{"error": "invalid S3 ID"})
		return
	}

	key := c.Query("key")
	if key == "" {
		c.JSON(400, gin.H{"error": "missing key parameter"})
		return
	}

	data, err := s3.Get(key)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.Data(200, "application/octet-stream", data)
}

func (gw *S3Gateway) handlePut(c *gin.Context) {
	s3 := gw.S3Map[c.GetString("s3_id")]
	if s3 == nil {
		c.JSON(400, gin.H{"error": "invalid S3 ID"})
		return
	}

	key := c.Query("key")
	if key == "" {
		c.JSON(400, gin.H{"error": "missing key parameter"})
		return
	}

	data, err := c.GetRawData()
	if err != nil {
		c.JSON(400, gin.H{"error": "failed to read request body"})
		return
	}

	if err := s3.Put(key, data); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.String(200, "OK")
}

func (gw *S3Gateway) handleList(c *gin.Context) {
	s3 := gw.S3Map[c.GetString("s3_id")]
	if s3 == nil {
		c.JSON(400, gin.H{"error": "invalid S3 ID"})
		return
	}

	prefix := c.Query("prefix")
	keys, err := s3.List(prefix)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, keys)
}

func (gw *S3Gateway) handleDelete(c *gin.Context) {
	s3 := gw.S3Map[c.GetString("s3_id")]
	if s3 == nil {
		c.JSON(400, gin.H{"error": "invalid S3 ID"})
		return
	}

	key := c.Query("key")
	if key == "" {
		c.JSON(400, gin.H{"error": "missing key parameter"})
		return
	}

	if err := s3.Delete(key); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.String(200, "OK")
}

// Reload 重新加载 S3 配置
func (gw *S3Gateway) Reload(newConf []common.S3Config) error {
	newIDs := make(map[string]common.S3Config)
	for _, conf := range newConf {
		newIDs[conf.ID] = conf
	}

	newS3Map := make(map[string]common.S3)

	// 保留未变化的 S3
	for id, s3 := range gw.S3Map {
		if newConf, exists := newIDs[id]; exists {
			oldConf := gw.findConfigByID(id)
			if oldConf != nil && oldConf.URL == newConf.URL {
				newS3Map[id] = s3
				common.Logger.Infof("S3 %s unchanged", id)
				delete(newIDs, id)
				continue
			}
		}
		if err := s3.Close(); err != nil {
			common.Logger.Warnf("Failed to close S3 %s: %v", id, err)
		}
		common.Logger.Infof("Closed S3 %s", id)
	}

	// 加载新增或变化的 S3
	for id, conf := range newIDs {
		parsed, err := ParseS3URL(conf.URL)
		if err != nil {
			common.Logger.Errorf("Failed to parse S3 URL for %s: %v", id, err)
			return err
		}

		s3, err := CreateS3(parsed)
		if err != nil {
			common.Logger.Errorf("Failed to create S3 %s: %v", id, err)
			return err
		}

		if err = s3.Start(); err != nil {
			common.Logger.Errorf("Failed to start S3 %s: %v", id, err)
			return err
		}

		newS3Map[id] = s3
		common.Logger.Infof("Loaded %s S3: %s", parsed.Type, id)
	}

	gw.S3Map = newS3Map
	gw.S3Conf = newConf
	return nil
}

func (gw *S3Gateway) findConfigByID(id string) *common.S3Config {
	for _, conf := range gw.S3Conf {
		if conf.ID == id {
			return &conf
		}
	}
	return nil
}

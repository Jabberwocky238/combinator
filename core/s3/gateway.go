package s3

import (
	"io"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
	"jabberwocky238/combinator/core/common/models"
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
		// 对象操作 - 使用 JSON body 传递参数
		gw.grg.POST("/head", gw.handleHead)
		gw.grg.POST("/get", gw.handleGet)
		gw.grg.POST("/put", gw.handlePut)
		gw.grg.POST("/delete", gw.handleDelete)
		gw.grg.POST("/copy", gw.handleCopy)
		gw.grg.POST("/list", gw.handleList)
		gw.grg.POST("/presigned-download-url", gw.handleGetPresignedURL)
		gw.grg.POST("/presigned-upload-url", gw.handlePutPresignedURL)
	}

	// 特殊路由：直接访问静态资源，不需要 middlewareS3
	gw.grg.GET("/-/:s3_id/*key", gw.handleStaticResource)

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

		// 从 header 获取 object key 并存储到 context
		key := c.GetHeader("X-Combinator-S3-Object-Key")
		if key != "" {
			c.Set("object_key", key)
		}

		c.Next()
	}
}

// handleHead 获取对象元数据
func (gw *S3Gateway) handleHead(c *gin.Context) {
	s3 := gw.S3Map[c.GetString("s3_id")]
	if s3 == nil {
		c.JSON(400, gin.H{"error": "invalid S3 ID"})
		return
	}

	// 优先从 body 获取 key，其次从 context（middleware 设置）
	var req struct {
		Key string `json:"key"`
	}
	c.ShouldBindJSON(&req)

	key := req.Key
	if key == "" {
		key = c.GetString("object_key")
	}

	if key == "" {
		c.JSON(400, gin.H{"error": "missing key parameter"})
		return
	}

	info, err := s3.Head(key)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, info)
}

func (gw *S3Gateway) handleGet(c *gin.Context) {
	s3 := gw.S3Map[c.GetString("s3_id")]
	if s3 == nil {
		c.JSON(400, gin.H{"error": "invalid S3 ID"})
		return
	}

	// 优先从 body 获取 key，其次从 context（middleware 设置）
	var req struct {
		Key string `json:"key"`
	}
	c.ShouldBindJSON(&req)

	key := req.Key
	if key == "" {
		key = c.GetString("object_key")
	}

	if key == "" {
		c.JSON(400, gin.H{"error": "missing key parameter"})
		return
	}

	var opts *models.S3GetOptions
	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		var start, end int64
		if _, err := strconv.ParseInt(rangeHeader, 10, 64); err == nil {
			opts = &models.S3GetOptions{
				Range: &models.S3Range{Start: start, End: end},
			}
		}
	}

	reader, info, err := s3.Get(key, opts)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()

	// 设置响应头
	if info.ContentType != "" {
		c.Header("Content-Type", info.ContentType)
	}
	c.Header("Content-Length", strconv.FormatInt(info.Size, 10))
	c.Header("Last-Modified", info.LastModified.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	if info.ETag != "" {
		c.Header("ETag", info.ETag)
	}

	c.Stream(func(w io.Writer) bool {
		io.Copy(w, reader)
		return false
	})
}

func (gw *S3Gateway) handlePut(c *gin.Context) {
	s3 := gw.S3Map[c.GetString("s3_id")]
	if s3 == nil {
		c.JSON(400, gin.H{"error": "invalid S3 ID"})
		return
	}

	// 从 context 获取 key（middleware 设置）
	key := c.GetString("object_key")
	if key == "" {
		c.JSON(400, gin.H{"error": "missing key parameter"})
		return
	}

	opts := &models.S3PutOptions{
		ContentType: c.GetHeader("Content-Type"),
	}

	size := c.Request.ContentLength
	if err := s3.Put(key, c.Request.Body, size, opts); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.String(200, "OK")
}

func (gw *S3Gateway) handleDelete(c *gin.Context) {
	s3 := gw.S3Map[c.GetString("s3_id")]
	if s3 == nil {
		c.JSON(400, gin.H{"error": "invalid S3 ID"})
		return
	}

	// 尝试解析 JSON body
	var opts models.S3DeleteOptions
	c.ShouldBindJSON(&opts)

	// 如果 body 为空，从 context 获取 key（middleware 设置）
	if len(opts.Keys) == 0 {
		key := c.GetString("object_key")
		if key == "" {
			c.JSON(400, gin.H{"error": "missing key parameter"})
			return
		}
		opts.Keys = []models.S3DeleteKey{
			{Mode: models.S3DeleteModePrecise, Key: key},
		}
	}

	deletedCount, err := s3.Delete(&opts)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"deleted": deletedCount})
}

// handleCopy 复制对象
func (gw *S3Gateway) handleCopy(c *gin.Context) {
	s3 := gw.S3Map[c.GetString("s3_id")]
	if s3 == nil {
		c.JSON(400, gin.H{"error": "invalid S3 ID"})
		return
	}

	var req struct {
		SrcKey string `json:"src_key"`
		DstKey string `json:"dst_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	if req.SrcKey == "" || req.DstKey == "" {
		c.JSON(400, gin.H{"error": "missing src_key or dst_key parameter"})
		return
	}

	if err := s3.Copy(req.SrcKey, req.DstKey); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.String(200, "OK")
}

// handleList 列出对象
func (gw *S3Gateway) handleList(c *gin.Context) {
	s3 := gw.S3Map[c.GetString("s3_id")]
	if s3 == nil {
		c.JSON(400, gin.H{"error": "invalid S3 ID"})
		return
	}

	var req struct {
		Prefix     string `json:"prefix"`
		MaxKeys    int    `json:"max_keys"`
		StartAfter string `json:"start_after"`
	}
	c.ShouldBindJSON(&req)

	opts := &models.S3ListOptions{
		Prefix:     req.Prefix,
		MaxKeys:    req.MaxKeys,
		StartAfter: req.StartAfter,
	}

	result, err := s3.List(opts)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

// handleGetPresignedURL 获取预签名下载URL
func (gw *S3Gateway) handleGetPresignedURL(c *gin.Context) {
	s3 := gw.S3Map[c.GetString("s3_id")]
	if s3 == nil {
		c.JSON(400, gin.H{"error": "invalid S3 ID"})
		return
	}

	var req struct {
		Key     string `json:"key"`
		Expires string `json:"expires,omitempty"` // 例如: "1h", "30m"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	if req.Key == "" {
		c.JSON(400, gin.H{"error": "missing key parameter"})
		return
	}

	expires := 1 * time.Hour
	if req.Expires != "" {
		if d, err := time.ParseDuration(req.Expires); err == nil {
			expires = d
		}
	}

	url, err := s3.GetPresignedURL(req.Key, expires)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"url": url})
}

// handlePutPresignedURL 获取预签名上传URL
func (gw *S3Gateway) handlePutPresignedURL(c *gin.Context) {
	s3 := gw.S3Map[c.GetString("s3_id")]
	if s3 == nil {
		c.JSON(400, gin.H{"error": "invalid S3 ID"})
		return
	}

	var req struct {
		Key     string `json:"key"`
		Expires string `json:"expires,omitempty"` // 例如: "1h", "30m"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	if req.Key == "" {
		c.JSON(400, gin.H{"error": "missing key parameter"})
		return
	}

	expires := 1 * time.Hour
	if req.Expires != "" {
		if d, err := time.ParseDuration(req.Expires); err == nil {
			expires = d
		}
	}

	url, err := s3.PutPresignedURL(req.Key, expires)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"url": url})
}

// handleStaticResource 直接访问静态资源
func (gw *S3Gateway) handleStaticResource(c *gin.Context) {
	s3ID := c.Param("s3_id")
	if s3ID == "" {
		c.Status(404)
		return
	}

	s3 := gw.S3Map[s3ID]
	if s3 == nil {
		c.Status(404)
		return
	}

	key := c.Param("key")
	if key == "" || key == "/" {
		c.Status(404)
		return
	}
	key = key[1:] // 移除开头的斜杠

	reader, info, err := s3.Get(key, nil)
	if err != nil {
		c.Status(404)
		return
	}
	defer reader.Close()

	// 设置响应头（从 Get 返回的元数据中获取）
	if info.ContentType != "" {
		c.Header("Content-Type", info.ContentType)
	}
	c.Header("Content-Length", strconv.FormatInt(info.Size, 10))
	c.Header("Last-Modified", info.LastModified.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	if info.ETag != "" {
		c.Header("ETag", info.ETag)
	}

	c.Stream(func(w io.Writer) bool {
		io.Copy(w, reader)
		return false
	})
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

package s3

import (
	"io"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
	"jabberwocky238/combinator/core/common/models"
)

var ConditionalS3StaticHandler func(map[string]common.S3) func(c *gin.Context)

type S3Gateway struct {
	*common.BaseGateway[common.S3, common.S3Config]
	grg           *gin.RouterGroup
	TenantHandler gin.HandlerFunc
}

func NewGateway(grg *gin.RouterGroup, conf []common.S3Config) *S3Gateway {
	parser := func(c common.S3Config) (common.S3, error) {
		parsed, err := ParseS3URL(c.URL)
		if err != nil {
			return nil, err
		}
		return CreateS3(parsed)
	}

	return &S3Gateway{
		BaseGateway: common.NewBaseGateway[common.S3, common.S3Config](conf, parser),
		grg:         grg,
	}
}

func (gw *S3Gateway) Start() error {
	// 使用 ModifyConfig 加载初始配置
	if err := gw.ModifyConfig(gw.InitConf, nil); err != nil {
		return err
	}

	gw.grg.Use(gw.middlewareS3())
	if gw.TenantHandler != nil {
		gw.grg.Use(gw.TenantHandler)
	}
	gw.grg.Use(gw.middlewareCatchS3())
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
	if ConditionalS3StaticHandler != nil {
		gw.grg.GET("/-/:s3_id/*key", ConditionalS3StaticHandler(gw.GetAllServices()))
	}

	return nil
}

func (gw *S3Gateway) Close() error {
	for _, s3 := range gw.GetAllServices() {
		s3.Close()
	}
	return nil
}

func (gw *S3Gateway) Type() string {
	return "s3-gateway"
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

func (gw *S3Gateway) middlewareCatchS3() gin.HandlerFunc {
	return func(c *gin.Context) {
		s3ID := c.MustGet("s3_id").(string)
		// 获取 S3 实例
		s3, ok := gw.Get(s3ID)
		if !ok {
			c.JSON(400, gin.H{"error": "invalid S3 ID"})
			c.Abort()
			return
		}
		c.Set("s3", s3)
		c.Next()
	}
}

// handleHead 获取对象元数据
func (gw *S3Gateway) handleHead(c *gin.Context) {
	s3 := c.MustGet("s3").(common.S3)

	// 只从 context（middleware 设置的 header）获取 key
	key := c.GetString("object_key")
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
	s3 := c.MustGet("s3").(common.S3)

	// 只从 context（middleware 设置的 header）获取 key
	key := c.GetString("object_key")
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
	s3 := c.MustGet("s3").(common.S3)

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
	s3 := c.MustGet("s3").(common.S3)

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
	s3 := c.MustGet("s3").(common.S3)

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
	s3 := c.MustGet("s3").(common.S3)

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
	s3 := c.MustGet("s3").(common.S3)

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
	s3 := c.MustGet("s3").(common.S3)

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

// Reload 重新加载 S3 配置

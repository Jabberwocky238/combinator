//go:build !prod
// +build !prod

// WARNING: dangerous endpoint should not be exposed in production environment.

package s3

import (
	"io"
	common "jabberwocky238/combinator/core/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

func init() {
	// 设置默认的静态资源处理函数，可以被外部覆盖
	ConditionalS3StaticHandler = func(S3Map map[string]common.S3) func(c *gin.Context) {
		return func(c *gin.Context) {
			handleStaticResource(c, S3Map)
		}
	}
}

// handleStaticResource 直接访问静态资源
func handleStaticResource(c *gin.Context, S3Map map[string]common.S3) {
	s3ID := c.Param("s3_id")
	if s3ID == "" {
		c.Status(404)
		return
	}

	s3 := S3Map[s3ID]
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

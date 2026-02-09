package combinator

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml/internal/errors"

	common "jabberwocky238/combinator/core/common"
	kvModule "jabberwocky238/combinator/core/kv"
	rdbModule "jabberwocky238/combinator/core/rdb"
	s3Module "jabberwocky238/combinator/core/s3"
)

type userAuthedResource struct {
	ResourceID   string `json:"resource_id"`
	ResourceType string `json:"resource_type"`
}

type Gateway struct {
	g               *gin.Engine
	ctx             context.Context
	userLock        sync.RWMutex
	userCredentials map[string]string
	userResources   map[string][]userAuthedResource
	rdbGateway      *rdbModule.RDBGateway
	kvGateway       *kvModule.KVGateway
	s3Gateway       *s3Module.S3Gateway
}

func NewGateway(confIn *common.Config, debug bool) *Gateway {
	conf := confIn
	r := gin.New()
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health"},
	}))
	r.Use(gin.Recovery())
	ctx := context.Background()

	if debug {
		openGatewayCors(r)
	} else {
		ctx = context.WithValue(ctx, "rdb_id_resolver", func(c *gin.Context) (string, error) {
			id := c.GetHeader("X-Combinator-RDB-ID")
			uid := c.GetString("user_id")
			if id == "" || uid == "" {
				return "", errors.New("Missing X-Combinator-RDB-ID header or user_id in context")
			}

		})
	}
	// Health check endpoint, 不打印日志
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "combinator",
		})
	})

	return &Gateway{
		g:          r,
		ctx:        ctx,
		rdbGateway: rdbModule.NewGateway(ctx, r.Group("/rdb"), conf.Rdb),
		kvGateway:  kvModule.NewGateway(r.Group("/kv"), conf.Kv),
		s3Gateway:  s3Module.NewGateway(r.Group("/s3"), conf.S3),
	}
}

func openGatewayCors(r *gin.Engine) {
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Combinator-RDB-ID, X-Combinator-KV-ID, X-Combinator-KV-Key, X-Combinator-S3-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
	})

	// options
	r.OPTIONS("/*cors", func(c *gin.Context) {
		c.AbortWithStatus(204)
	})
}

func (gw *Gateway) Start(addr string) error {
	gw.g.GET("/", func(c *gin.Context) {
		// text and timestamp
		timestamp := time.Now().Format(time.RFC3339)
		c.String(http.StatusOK, "Combinator Service is running at %s. Serving for %d users", timestamp, len(gw.userCredentials))
	})

	err := gw.rdbGateway.Start()
	if err != nil {
		return err
	}

	err = gw.kvGateway.Start()
	if err != nil {
		return err
	}

	err = gw.s3Gateway.Start()
	if err != nil {
		return err
	}

	return gw.g.Run(addr)
}

type returntype struct {
	SecretKey string               `json:"secret_key,omitempty"`
	Resources []userAuthedResource `json:"resources,omitempty"`
}

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

func (gw *Gateway) SetupVerify() {
	gw.g.Use(func(c *gin.Context) {
		signature := c.GetHeader("X-Raysail-Signature")
		userID := c.GetHeader("X-Raysail-UID")
		if signature == "" || userID == "" {
			c.AbortWithStatusJSON(400, gin.H{
				"error": "Missing X-Raysail-Signature or X-Raysail-UID header",
			})
			return
		}
		c.Set("user_id", userID)
		c.Set("signature", signature)
		gw.userLock.RLock()
		secretKey, ok := gw.userCredentials[userID]
		gw.userLock.RUnlock()
		if !ok {
			// 发起查询
			// 向http://control-plane.console.svc.cluster.local:9901/combinator/retrieveSecretByID
			// 查询 userID 是否存在
			res, err := httpClient.Get("http://control-plane.console.svc.cluster.local:9901/combinator/retrieveSecretByID?user_id=" + userID)
			if err != nil || res.StatusCode != 200 {
				c.AbortWithStatusJSON(500, gin.H{
					"error": "Internal error: failed to retrieve secret",
				})
				return
			}
			var result returntype
			if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
				c.AbortWithStatusJSON(500, gin.H{
					"error": "Internal error: failed to parse secret response",
				})
				return
			}
			gw.userLock.Lock()
			gw.userCredentials[userID] = result.SecretKey
			gw.userResources[userID] = result.Resources
			gw.userLock.Unlock()
			secretKey = result.SecretKey
		}
		// 使用hmac计算signature
		// 获取body
		body, err := c.GetRawData()
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{
				"error": "Failed to read request body",
			})
			return
		}
		if err := common.VerifyHMACSignature(secretKey, body, signature); err != nil {
			c.AbortWithStatusJSON(401, gin.H{
				"error": "Invalid signature",
			})
			return
		}
		// 将body重新写回请求中，以便后续处理
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		c.Next()
	})
}

package combinator

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
	kvModule "jabberwocky238/combinator/core/kv"
	rdbModule "jabberwocky238/combinator/core/rdb"
	s3Module "jabberwocky238/combinator/core/s3"
)

type Gateway struct {
	g             *gin.Engine
	ig            *gin.Engine
	ctx           context.Context
	rdbGateway    *rdbModule.RDBGateway
	kvGateway     *kvModule.KVGateway
	s3Gateway     *s3Module.S3Gateway
	tenantManager *MultiTenantManager
}

func NewGateway(confIn *common.Config, debug bool) *Gateway {
	r := gin.New()
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health"},
	}))
	r.Use(gin.Recovery())
	// Health check endpoint, 不打印日志
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "combinator",
		})
	})

	rdbGateway := rdbModule.NewGateway(r.Group("/rdb"), confIn.Rdb)
	kvGateway := kvModule.NewGateway(r.Group("/kv"), confIn.Kv)
	s3Gateway := s3Module.NewGateway(r.Group("/s3"), confIn.S3)

	var tenantManager *MultiTenantManager
	var ctx context.Context = context.Background()
	var ig *gin.Engine
	if debug {
		openGatewayCors(r)
	} else {
		// 创建多租户管理器
		tenantManager = NewMultiTenantManager(ctx, rdbGateway, kvGateway, s3Gateway)
		ig = gin.New() // 内部使用的 Engine，不对外暴露
		ig.POST("/webhook", func(ctx *gin.Context) {
			var req struct {
				UserUID      string `json:"user_uid"`
				ResourceID   string `json:"resource_id"`
				ResourceType string `json:"resource_type"` // "rdb", "kv", "s3"
			}
			if err := ctx.ShouldBindJSON(&req); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
				return
			}
			err := tenantManager.DeleteTenantResource(req.UserUID, req.ResourceType, req.ResourceID)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"status": "success"})
		})
	}

	return &Gateway{
		g:             r,
		ig:            ig,
		ctx:           ctx,
		rdbGateway:    rdbGateway,
		kvGateway:     kvGateway,
		s3Gateway:     s3Gateway,
		tenantManager: tenantManager,
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
	// 启用多租户中间件
	if gw.tenantManager != nil {
		gw.g.Use(gw.tenantManager.Middleware())
	}

	gw.g.GET("/", func(c *gin.Context) {
		// text and timestamp
		timestamp := time.Now().Format(time.RFC3339)
		if gw.tenantManager == nil {
			c.String(http.StatusOK, "Combinator Service is running at %s. Multi-tenancy is disabled.", timestamp)
			return
		}
		// no need to precisely count tenants, just return the length of the map
		tenantCount := len(gw.tenantManager.tenants)
		c.String(http.StatusOK, "Combinator Service is running at %s. Serving for %d tenants", timestamp, tenantCount)
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

	go func() {
		if gw.ig != nil {
			err := gw.ig.Run("0.0.0.0:8890")
			if err != nil {
				panic(err)
			}
		}
	}()

	return gw.g.Run(addr)
}

func (gw *Gateway) Close() error {
	if err := gw.rdbGateway.Close(); err != nil {
		return err
	}
	if err := gw.kvGateway.Close(); err != nil {
		return err
	}
	if err := gw.s3Gateway.Close(); err != nil {
		return err
	}
	return nil
}

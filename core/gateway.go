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
	ctx           context.Context
	rdbGateway    *rdbModule.RDBGateway
	kvGateway     *kvModule.KVGateway
	s3Gateway     *s3Module.S3Gateway
	tenantManager *MultiTenantManager
}

func NewGateway(confIn *common.Config, debug bool) *Gateway {
	r := gin.New()
	r.Use(gin.Recovery())

	var tenantManager *MultiTenantManager
	var ctx context.Context = context.Background()
	if debug {
		openGatewayCors(r)
	} else {
		// 创建多租户管理器并设置 TenantHandler
		tenantManager = NewMultiTenantManager(ctx)
		// 在路由组上注册全局中间件（必须在 gateway.Start() 之前）
		r.Use(tenantManager.Middleware())
	}

	// 创建路由组
	rdbGateway := rdbModule.NewGateway(r.Group("/rdb"), confIn.Rdb)
	kvGateway := kvModule.NewGateway(r.Group("/kv"), confIn.Kv)
	s3Gateway := s3Module.NewGateway(r.Group("/s3"), confIn.S3)

	if tenantManager != nil {
		tenantManager.SetupMiddleware(rdbGateway, kvGateway, s3Gateway)
	}

	return &Gateway{
		g:             r,
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
	// 启动多租户管理器的独立服务
	if gw.tenantManager != nil {
		go gw.tenantManager.Start("0.0.0.0:8890")
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

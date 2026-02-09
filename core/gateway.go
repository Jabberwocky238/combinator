package combinator

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
	kvModule "jabberwocky238/combinator/core/kv"
	rdbModule "jabberwocky238/combinator/core/rdb"
	s3Module "jabberwocky238/combinator/core/s3"
)

type Gateway struct {
	g                   *gin.Engine
	userCredentialsLock sync.RWMutex
	userCredentials     map[string]string
	rdbGateway          *rdbModule.RDBGateway
	kvGateway           *kvModule.KVGateway
	s3Gateway           *s3Module.S3Gateway
}

func NewGateway(confIn *common.Config, debug bool) *Gateway {
	conf := confIn
	r := gin.New()
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health"},
	}))
	r.Use(gin.Recovery())

	if debug {
		openGatewayCors(r)
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
		rdbGateway: rdbModule.NewGateway(r.Group("/rdb"), conf.Rdb),
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

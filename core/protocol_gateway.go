package combinator

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
	kvModule "jabberwocky238/combinator/core/kv"
	rdbModule "jabberwocky238/combinator/core/rdb"
)

type Gateway struct {
	g          *gin.Engine
	rdbGateway *rdbModule.RDBGateway
	kvGateway  *kvModule.KVGateway
}

func NewGateway(confIn *common.Config) *Gateway {
	conf, err := configCheck(confIn)
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.Use(gin.Recovery())
	r.GET("/", func(c *gin.Context) {
		// text and timestamp
		timestamp := time.Now().Format(time.RFC3339)
		c.String(http.StatusOK, "Combinator Service is running at %s.", timestamp)
	})
	// Health check endpoint
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
	}
}

func configCheck(confs *common.Config) (common.Config, error) {
	var resConf common.Config
	for _, rdbConf := range confs.Rdb {
		if !rdbConf.Enabled {
			continue
		}
		resConf.Rdb = append(resConf.Rdb, rdbConf)
	}
	for _, kvConf := range confs.Kv {
		if !kvConf.Enabled {
			continue
		}
		resConf.Kv = append(resConf.Kv, kvConf)
	}
	return resConf, nil
}

func (gw *Gateway) Start(addr string) error {
	err := gw.rdbGateway.Start()
	if err != nil {
		return err
	}

	err = gw.kvGateway.Start()
	if err != nil {
		return err
	}

	return gw.g.Run(addr)
}

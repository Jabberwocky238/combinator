package combinator

import (
	"encoding/json"
	"fmt"
	"io"
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

func NewGateway(confIn *common.Config, cors bool) *Gateway {
	conf, err := configCheck(confIn)
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.Use(gin.Recovery())
	if cors {
		openGatewayCors(r)
	}
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

func openGatewayCors(r *gin.Engine) {
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Combinator-RDB-ID, X-Combinator-KV-ID, X-Combinator-KV-Key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
	})

	// options
	r.OPTIONS("/*cors", func(c *gin.Context) {
		c.AbortWithStatus(204)
	})
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

// Reload 重新加载配置
func (gw *Gateway) Reload(confIn *common.Config) error {
	conf, err := configCheck(confIn)
	if err != nil {
		return err
	}

	// 重新加载 RDB Gateway
	if err := gw.rdbGateway.Reload(conf.Rdb); err != nil {
		return err
	}

	// 重新加载 KV Gateway
	if err := gw.kvGateway.Reload(conf.Kv); err != nil {
		return err
	}

	return nil
}

// API 监听
func (gw *Gateway) SetupReloadAPI(reloadChan chan<- *common.Config) {
	gw.g.POST("/reload", func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.JSON(405, gin.H{"error": "Method not allowed"})
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(400, gin.H{"error": "Failed to read body"})
			return
		}

		var config common.Config
		if err := json.Unmarshal(body, &config); err != nil {
			c.JSON(400, gin.H{"error": "Invalid JSON"})
			return
		}

		fmt.Println("🔄 Received reload request via API...")
		reloadChan <- &config
		c.String(200, "Config Reloaded")
	})
}

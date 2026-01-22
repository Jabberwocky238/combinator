package rdb

import (
	"io"

	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
)

type RDBGateway struct {
	grg     *gin.RouterGroup
	rdbConf []common.RDBConfig
	rdbMap  map[string]common.RDB
}

func NewGateway(grg *gin.RouterGroup, conf []common.RDBConfig) *RDBGateway {
	return &RDBGateway{
		grg:     grg,
		rdbConf: conf,
		rdbMap:  make(map[string]common.RDB),
	}
}

func (gw *RDBGateway) loadRDBs() error {
	var err error
	for _, rdbConf := range gw.rdbConf {
		parsed, err := ParseRDBURL(rdbConf.URL)
		switch parsed.Type {
		case "postgres":
			gw.rdbMap[rdbConf.ID] = NewPsqlRDB(parsed.Host, parsed.Port, parsed.User, parsed.Password, parsed.DBName)
		case "sqlite":
			gw.rdbMap[rdbConf.ID] = NewSqliteRDB(parsed.Path)
		default:
			return err
		}
		if err = gw.rdbMap[rdbConf.ID].Start(); err != nil {
			return err
		}
		common.Logger.Infof("Loaded %s RDB: %s", parsed.Type, rdbConf.ID)
	}
	return err
}

func (gw *RDBGateway) Start() error {
	err := gw.loadRDBs()
	if err != nil {
		return err
	}

	// RDB 路由组
	gw.grg.Use(middlewareRDB())
	{
		gw.grg.POST("/query", gw.handleQuery)
		gw.grg.POST("/exec", handleExec)
		gw.grg.POST("/batch", handleBatch)
	}

	return nil
}

func middlewareRDB() gin.HandlerFunc {
	return func(c *gin.Context) {
		rdbID := c.GetHeader("X-Combinator-RDB-ID")
		if rdbID == "" {
			c.JSON(400, gin.H{"error": "missing X-Combinator-RDB-ID header"})
			c.Abort()
			return
		}

		// 注入 RDB ID 到 context
		c.Set("rdb_id", rdbID)
		c.Next()
	}
}

type RDBQueryRequest struct {
	Stmt string `json:"stmt"`
	Args []any  `json:"args"`
}

func (gw *RDBGateway) handleQuery(c *gin.Context) {
	rdbId, _ := c.Get("rdb_id")

	rdb := gw.rdbMap[rdbId.(string)]
	if rdb == nil {
		c.JSON(400, gin.H{"error": "invalid RDB ID"})
		return
	}

	// 解析请求体
	var req RDBQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	// 设置响应头为 CSV 流式输出
	c.Header("Content-Type", "text/csv")
	c.Header("Transfer-Encoding", "chunked")
	c.Status(200)

	// 流式写入 CSV 数据
	writer := c.Writer

	// 示例：写入 CSV 头和数据
	io.WriteString(writer, "id,name,value\n")
	writer.Flush()

	// TODO: 从数据库结果集逐行写入
	io.WriteString(writer, "1,test,100\n")
	writer.Flush()
}

func handleExec(c *gin.Context) {
	// TODO: 处理执行请求
	c.JSON(200, gin.H{"message": "exec endpoint"})
}

func handleBatch(c *gin.Context) {
	// TODO: 处理批量请求
	c.JSON(200, gin.H{"message": "batch endpoint"})
}

package rdb

import (
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
	for _, rdbConf := range gw.rdbConf {
		parsed, err := ParseRDBURL(rdbConf.URL)
		if err != nil {
			common.Logger.Errorf("Failed to parse RDB URL for %s: %v", rdbConf.ID, err)
			return err
		}

		switch parsed.Type {
		case "postgres":
			gw.rdbMap[rdbConf.ID] = NewPsqlRDB(parsed.Host, parsed.Port, parsed.User, parsed.Password, parsed.DBName)
		case "sqlite":
			gw.rdbMap[rdbConf.ID] = NewSqliteRDB(parsed.Path)
		default:
			common.Logger.Errorf("Unsupported RDB type: %s", parsed.Type)
			return err
		}

		if err = gw.rdbMap[rdbConf.ID].Start(); err != nil {
			common.Logger.Errorf("Failed to start RDB %s: %v", rdbConf.ID, err)
			return err
		}
		common.Logger.Infof("Loaded %s RDB: %s", parsed.Type, rdbConf.ID)
	}
	return nil
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
		gw.grg.POST("/exec", gw.handleExec)
		gw.grg.POST("/batch", gw.handleBatch)
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
	rdb := gw.rdbMap[c.GetString("rdb_id")]
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
	data, err := rdb.Query(req.Stmt, req.Args...)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Writer.Write(data)
}

type RDBExecRequest struct {
	Stmt string `json:"stmt"`
	Args []any  `json:"args"`
}

type RDBExecResponse struct {
	LastInsertId int `json:"last_insert_id"`
	RowsAffected int `json:"rows_affected"`
}

func (gw *RDBGateway) handleExec(c *gin.Context) {
	rdb := gw.rdbMap[c.GetString("rdb_id")]
	if rdb == nil {
		c.JSON(400, gin.H{"error": "invalid RDB ID"})
		return
	}

	var req RDBExecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	lastInsertId, rowsAffected, err := rdb.Execute(req.Stmt, req.Args...)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var resp RDBExecResponse
	resp.LastInsertId = lastInsertId
	resp.RowsAffected = rowsAffected

	c.JSON(200, resp)
}

func (gw *RDBGateway) handleBatch(c *gin.Context) {
	rdb := gw.rdbMap[c.GetString("rdb_id")]
	if rdb == nil {
		c.JSON(400, gin.H{"error": "invalid RDB ID"})
		return
	}

	// 直接解析 JSON 数组
	var stmtList []string
	if err := c.ShouldBindJSON(&stmtList); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	err := rdb.Batch(stmtList)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.String(200, "OK")
}

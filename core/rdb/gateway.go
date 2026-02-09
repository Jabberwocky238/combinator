package rdb

import (
	"io"

	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
)

var EB = common.GlobalErrorBuilder.With("rdb")

type RDBGateway struct {
	*common.BaseGateway[common.RDB, common.RDBConfig]
	grg           *gin.RouterGroup
	TenantHandler gin.HandlerFunc
}

func NewGateway(grg *gin.RouterGroup, conf []common.RDBConfig) *RDBGateway {
	parser := func(c common.RDBConfig) (common.RDB, error) {
		parsed, err := ParseRDBURL(c.URL)
		if err != nil {
			return nil, err
		}
		return CreateRDB(parsed)
	}

	gw := RDBGateway{
		BaseGateway: common.NewBaseGateway(conf, parser),
		grg:         grg,
	}
	return &gw
}

func (gw *RDBGateway) Start() error {
	// 使用 ModifyConfig 加载初始配置
	if err := gw.ModifyConfig(gw.InitConf, nil); err != nil {
		return err
	}

	// 设置路由
	gw.grg.Use(gw.middlewareRDB())
	if gw.TenantHandler != nil {
		gw.grg.Use(gw.TenantHandler)
	}
	gw.grg.Use(gw.middlewareCatchRDB())
	{
		gw.grg.POST("/query", gw.handleQuery)
		gw.grg.POST("/exec", gw.handleExec)
		gw.grg.POST("/batch", gw.handleBatch)
	}
	return nil
}

func (gw *RDBGateway) Close() error {
	for _, rdb := range gw.GetAllServices() {
		rdb.Close()
	}
	return nil
}

func (gw *RDBGateway) Type() string {
	return "rdb-gateway"
}

func (gw *RDBGateway) middlewareRDB() gin.HandlerFunc {
	return func(c *gin.Context) {
		rdbID := c.GetHeader("X-Combinator-RDB-ID")
		if rdbID == "" {
			c.JSON(400, gin.H{"error": "missing X-Combinator-RDB-ID header"})
			c.Abort()
			return
		}

		c.Set("rdb_id", rdbID)
		c.Next()
	}
}

func (gw *RDBGateway) middlewareCatchRDB() gin.HandlerFunc {
	return func(c *gin.Context) {
		rdbID := c.MustGet("rdb_id").(string)
		rdb, ok := gw.Get(rdbID)
		if !ok {
			c.JSON(400, gin.H{"error": "invalid RDB ID"})
			c.Abort()
			return
		}
		c.Set("rdb", rdb)
		c.Next()
	}
}

type RDBQueryRequest struct {
	Stmt string `json:"stmt"`
	Args []any  `json:"args"`
}

func (gw *RDBGateway) handleQuery(c *gin.Context) {
	rdb := c.MustGet("rdb").(common.RDB)

	// 解析请求体
	var req RDBQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	// 设置响应头为 CSV 流式输出
	reader, err := rdb.Query(req.Stmt, req.Args...)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "application/csv")
	c.Status(200)
	_, err = io.Copy(c.Writer, reader)
	reader.Close()
}

type RDBExecRequest struct {
	Stmt string `json:"stmt"`
	Args []any  `json:"args"`
}

func (gw *RDBGateway) handleExec(c *gin.Context) {
	rdb := c.MustGet("rdb").(common.RDB)

	var req RDBExecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	err := rdb.Exec(req.Stmt, req.Args...)
	if err != nil {
		common.Logger.Errorf("Execute failed: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.String(200, "OK")
}

type RDBBatchRequest []RDBExecRequest

func (gw *RDBGateway) handleBatch(c *gin.Context) {
	rdb := c.MustGet("rdb").(common.RDB)

	// 直接解析 JSON 数组
	var reqBody RDBBatchRequest
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	common.Logger.Debugf("Executing batch of %d statements", len(reqBody))
	var stmts []string
	var args [][]any
	for _, req := range reqBody {
		stmts = append(stmts, req.Stmt)
		args = append(args, req.Args)
	}
	err := rdb.Batch(stmts, args)
	if err != nil {
		common.Logger.Errorf("Batch execution failed: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.String(200, "OK")
}

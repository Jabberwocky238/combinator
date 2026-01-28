package rdb

import (
	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
)

var EB = common.GlobalErrorBuilder.With("rdb")

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

func (gw *RDBGateway) Start() error {
	err := gw.Reload(gw.rdbConf)
	if err != nil {
		return err
	}

	// RDB 路由组
	gw.grg.Use(gw.middlewareRDB())
	{
		gw.grg.POST("/query", gw.handleQuery)
		gw.grg.POST("/exec", gw.handleExec)
		gw.grg.POST("/batch", gw.handleBatch)
	}

	return nil
}

func (gw *RDBGateway) middlewareRDB() gin.HandlerFunc {
	return func(c *gin.Context) {
		rdbID := c.GetHeader("X-Combinator-RDB-ID")
		if rdbID == "" {
			c.JSON(400, gin.H{"error": "missing X-Combinator-RDB-ID header"})
			c.Abort()
			return
		}
		if gw.rdbMap[rdbID] == nil {
			c.JSON(400, gin.H{"error": "invalid RDB ID"})
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

	c.Header("Content-Type", "application/csv")
	c.Writer.Write(data)
}

type RDBExecRequest struct {
	Stmt string `json:"stmt"`
	Args []any  `json:"args"`
}

func (gw *RDBGateway) handleExec(c *gin.Context) {
	rdb := gw.rdbMap[c.GetString("rdb_id")]

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
	rdb := gw.rdbMap[c.GetString("rdb_id")]

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

func (gw *RDBGateway) Reload(newConf []common.RDBConfig) error {
	// 构建新配置的 ID 集合
	newIDs := make(map[string]common.RDBConfig)
	for _, conf := range newConf {
		newIDs[conf.ID] = conf
	}

	// 构建旧配置的 ID 集合
	oldIDs := make(map[string]bool)
	for _, conf := range gw.rdbConf {
		oldIDs[conf.ID] = true
	}

	// 创建新的 RDB map
	newRDBMap := make(map[string]common.RDB)

	// 1. 保留未变化的 RDB
	for id, rdb := range gw.rdbMap {
		if newConf, exists := newIDs[id]; exists {
			// 检查配置是否变化
			oldConf := gw.findConfigByID(id)
			if oldConf != nil && oldConf.URL == newConf.URL {
				// 配置未变化，保留
				newRDBMap[id] = rdb
				common.Logger.Infof("RDB %s unchanged, keeping connection", id)
				delete(newIDs, id)
				continue
			}
		}
		// 配置变化或被删除，关闭旧连接
		if err := rdb.Close(); err != nil {
			common.Logger.Warnf("Failed to close RDB %s: %v", id, err)
		}
		common.Logger.Infof("Closed RDB %s", id)
	}

	// 2. 加载新增或变化的 RDB
	for id, conf := range newIDs {
		parsed, err := ParseRDBURL(conf.URL)
		if err != nil {
			common.Logger.Errorf("Failed to parse RDB URL for %s: %v", id, err)
			return err
		}

		var rdb common.RDB
		switch parsed.Type {
		case "postgres":
			rdb = NewPsqlRDB(parsed.Host, parsed.Port, parsed.User, parsed.Password, parsed.DBName)
		case "sqlite":
			rdb = NewSqliteRDB(parsed.Path)
		default:
			common.Logger.Errorf("Unsupported RDB type: %s", parsed.Type)
			return err
		}

		if err = rdb.Start(); err != nil {
			common.Logger.Errorf("Failed to start RDB %s: %v", id, err)
			return err
		}

		newRDBMap[id] = rdb
		common.Logger.Infof("Loaded %s RDB: %s", parsed.Type, id)
	}

	// 3. 更新配置和 map
	gw.rdbMap = newRDBMap
	gw.rdbConf = newConf

	return nil
}

// findConfigByID 查找配置
func (gw *RDBGateway) findConfigByID(id string) *common.RDBConfig {
	for _, conf := range gw.rdbConf {
		if conf.ID == id {
			return &conf
		}
	}
	return nil
}

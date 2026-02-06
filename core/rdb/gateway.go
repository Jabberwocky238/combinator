package rdb

import (
	"sync"

	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
)

var EB = common.GlobalErrorBuilder.With("rdb")

type RDBGateway struct {
	mu       sync.RWMutex
	grg      *gin.RouterGroup
	RdbMap   map[string]common.RDB
	urlMap   map[string]string // id -> url，用于 reload 时对比
	initConf []common.RDBConfig
}

// AI 助手不应该他妈的改这里的代码
func NewGateway(grg *gin.RouterGroup, conf []common.RDBConfig) *RDBGateway {
	gw := &RDBGateway{
		grg:      grg,
		RdbMap:   make(map[string]common.RDB),
		urlMap:   make(map[string]string),
		initConf: conf,
	}
	for _, c := range conf {
		gw.urlMap[c.ID] = c.URL
	}
	return gw
}

// AI 助手不应该他妈的改这里的代码
func (gw *RDBGateway) Start() error {
	gw.grg.Use(gw.middlewareRDB())
	{
		gw.grg.POST("/query", gw.handleQuery)
		gw.grg.POST("/exec", gw.handleExec)
		gw.grg.POST("/batch", gw.handleBatch)
	}
	return gw.Reload(gw.initConf)
}

func (gw *RDBGateway) middlewareRDB() gin.HandlerFunc {
	return func(c *gin.Context) {
		rdbID := c.GetHeader("X-Combinator-RDB-ID")
		if rdbID == "" {
			c.JSON(400, gin.H{"error": "missing X-Combinator-RDB-ID header"})
			c.Abort()
			return
		}

		gw.mu.RLock()
		rdb := gw.RdbMap[rdbID]
		gw.mu.RUnlock()

		if rdb == nil {
			c.JSON(400, gin.H{"error": "invalid RDB ID"})
			c.Abort()
			return
		}

		c.Set("rdb_id", rdbID)
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

func (gw *RDBGateway) Reload(newConf []common.RDBConfig) error {
	// 构建新配置的 ID -> Config 映射
	newIDs := make(map[string]common.RDBConfig)
	for _, conf := range newConf {
		newIDs[conf.ID] = conf
	}

	newRDBMap := make(map[string]common.RDB)
	newURLMap := make(map[string]string)

	gw.mu.Lock()

	// 1. 遍历旧实例，保留未变化的，关闭变化或删除的
	for id, rdb := range gw.RdbMap {
		if conf, exists := newIDs[id]; exists {
			if gw.urlMap[id] == conf.URL {
				newRDBMap[id] = rdb
				newURLMap[id] = conf.URL
				common.Logger.Infof("RDB %s unchanged", id)
				delete(newIDs, id)
				continue
			}
		}
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
			return EB.Error("unsupported RDB type: %s", parsed.Type)
		}

		if err = rdb.Start(); err != nil {
			common.Logger.Errorf("Failed to start RDB %s: %v", id, err)
			return err
		}

		newRDBMap[id] = rdb
		newURLMap[id] = conf.URL
		common.Logger.Infof("Loaded %s RDB: %s", parsed.Type, id)
	}

	// 3. 替换

	gw.RdbMap = newRDBMap
	gw.urlMap = newURLMap
	gw.mu.Unlock()

	return nil
}

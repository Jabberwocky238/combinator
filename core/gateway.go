package combinator

import (
	"errors"
	"io"
	"net/http"

	"jabberwocky238/combinator/core/rdb"

	"github.com/gin-gonic/gin"
)

type Gateway struct {
	endpoint  string
	ginServer *gin.Engine
	processor *Processor
}

func NewGateway(endpoint string) *Gateway {
	processor := NewProcessor()

	// 默认添加一个 id为 "1" 的 sqlite RDB 用于测试
	processor.AddRDB("1", rdb.NewSqliteRDB(":memory:"))
	// processor.AddRDB("2", rdb.NewPsqlRDB("localhost", 5432, "combine1", "combine1", "combine1db"))

	return &Gateway{
		endpoint:  endpoint,
		processor: processor,
	}
}

func (g *Gateway) rdbHandler(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")
	rdbId := c.GetHeader("X-Combinator-RDB-ID")
	rpcMethod := c.GetHeader("X-Combinator-RPC-Method")
	if contentType != "application/sql" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content-Type must be application/sql"})
		return
	}
	if rdbId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id parameter is required"})
		return
	}

	// 从 processor 中查找对应的 RDB
	rdb, exists := g.processor.GetRDB(rdbId)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "RDB not found for id: " + rdbId})
		return
	}

	// 读取 SQL 语句
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	stmt := string(body)
	// 执行 SQL（所有逻辑在 RDB 层处理）
	var data []byte
	switch rpcMethod {
	case "Query":
		data, err = rdb.Query(stmt)
	case "Execute":
		data, err = rdb.Execute(stmt)
	case "Batch":
		err = rdb.Batch(stmt)
		data = []byte("OK")
	default:
		err = errors.New("")
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 使用 RDB 返回的 Content-Type
	c.Data(http.StatusOK, "combinator/rdb", data)
}

func (g *Gateway) Start() error {
	err := g.processor.Start()
	if err != nil {
		return err
	}

	r := gin.Default()
	g.ginServer = r

	r.POST("/rdb", g.rdbHandler)

	return r.Run(g.endpoint)
}

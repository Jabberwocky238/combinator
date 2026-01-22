package combinator

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

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

// RDBRequest represents a JSON request for prepared statements
type RDBRequest struct {
	Type   string   `json:"type"`   // "query" or "exec"
	Stmt   string   `json:"stmt"`   // SQL statement with ? placeholders
	Params []any    `json:"params"` // Parameters to fill placeholders
}

func (g *Gateway) rdbHandler(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")
	rdbId := c.GetHeader("X-Combinator-RDB-ID")

	if rdbId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Combinator-RDB-ID header is required"})
		return
	}

	// Find the RDB from processor
	rdb, exists := g.processor.GetRDB(rdbId)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "RDB not found for id: " + rdbId})
		return
	}

	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	var data []byte

	// Route based on Content-Type
	if contentType == "application/json" {
		// JSON request: parse and execute prepared statement
		data, err = g.handleJSONRequest(rdb, body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Return response with appropriate content type
		c.Data(http.StatusOK, "combinator/rdb", data)
	} else {
		// Text request: route to Batch for processing
		stmt := string(body)
		err = rdb.Batch(stmt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Batch returns no data, just success
		c.Data(http.StatusOK, "combinator/rdb", []byte("OK"))
	}
}

// handleJSONRequest parses JSON request and executes prepared statement
func (g *Gateway) handleJSONRequest(rdb RDB, body []byte) ([]byte, error) {
	var req RDBRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	// Validate request fields
	if req.Type == "" {
		return nil, errors.New("'type' field is required (must be 'query' or 'exec')")
	}
	if req.Stmt == "" {
		return nil, errors.New("'stmt' field is required")
	}

	// Validate that stmt contains placeholders
	if !strings.Contains(req.Stmt, "?") {
		return nil, errors.New("'stmt' must contain '?' placeholders for prepared statements")
	}

	// Validate params array exists
	if req.Params == nil {
		return nil, errors.New("'params' field is required (must be an array)")
	}

	// Execute based on type
	switch req.Type {
	case "query":
		return rdb.Query(req.Stmt, req.Params...)
	case "exec":
		return rdb.Execute(req.Stmt, req.Params...)
	default:
		return nil, fmt.Errorf("invalid type '%s': must be 'query' or 'exec'", req.Type)
	}
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

package combinator

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
	ID      any       `json:"id"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// API 监听
func (gw *Gateway) SetupMonitorAPI() {
	gw.g.POST("/monitor", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, JSONRPCResponse{
				JSONRPC: "2.0",
				Error:   &RPCError{Code: -32700, Message: "Parse error"},
				ID:      nil,
			})
			return
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(body, &req); err != nil {
			c.JSON(http.StatusOK, JSONRPCResponse{
				JSONRPC: "2.0",
				Error:   &RPCError{Code: -32700, Message: "Parse error"},
				ID:      nil,
			})
			return
		}

		if req.JSONRPC != "2.0" {
			c.JSON(http.StatusOK, JSONRPCResponse{
				JSONRPC: "2.0",
				Error:   &RPCError{Code: -32600, Message: "Invalid Request"},
				ID:      req.ID,
			})
			return
		}

		result, rpcErr := gw.handleRPCMethod(req.Method, req.Params)
		if rpcErr != nil {
			c.JSON(http.StatusOK, JSONRPCResponse{
				JSONRPC: "2.0",
				Error:   rpcErr,
				ID:      req.ID,
			})
			return
		}

		c.JSON(http.StatusOK, JSONRPCResponse{
			JSONRPC: "2.0",
			Result:  result,
			ID:      req.ID,
		})
	})
}

type ServiceInfo struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type ServiceListResult struct {
	RDB []ServiceInfo `json:"rdb"`
	KV  []ServiceInfo `json:"kv"`
}

func (gw *Gateway) handleRPCMethod(method string, params json.RawMessage) (any, *RPCError) {
	switch method {
	case "ping":
		return "pong", nil
	case "service.list":
		return gw.handleServiceList()
	default:
		return nil, &RPCError{Code: -32601, Message: "Method not found"}
	}
}

func (gw *Gateway) handleServiceList() (*ServiceListResult, *RPCError) {
	result := &ServiceListResult{
		RDB: make([]ServiceInfo, 0),
		KV:  make([]ServiceInfo, 0),
	}

	for id, rdb := range gw.rdbGateway.RdbMap {
		result.RDB = append(result.RDB, ServiceInfo{
			ID:   id,
			Type: rdb.Type(),
		})
	}

	for id, kv := range gw.kvGateway.KvMap {
		result.KV = append(result.KV, ServiceInfo{
			ID:   id,
			Type: kv.Type(),
		})
	}

	return result, nil
}

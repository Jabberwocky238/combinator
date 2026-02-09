package combinator

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	common "jabberwocky238/combinator/core/common"

	"github.com/gin-gonic/gin"
)

type returntype struct {
	SecretKey string `json:"secret_key,omitempty"`
	Resources []struct {
		ID   string `json:"resource_id"`
		Type string `json:"resource_type"`
	} `json:"resources,omitempty"`
}

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

func (gw *Gateway) SetupVerify() {
	gw.g.Use(func(c *gin.Context) {
		signature := c.GetHeader("X-Raysail-Signature")
		userID := c.GetHeader("X-Raysail-UID")
		if signature == "" || userID == "" {
			c.AbortWithStatusJSON(400, gin.H{
				"error": "Missing X-Raysail-Signature or X-Raysail-UID header",
			})
			return
		}
		gw.userCredentialsLock.RLock()
		secretKey, ok := gw.userCredentials[userID]
		gw.userCredentialsLock.RUnlock()
		if !ok {
			// 发起查询
			// 向http://control-plane.console.svc.cluster.local:9901/combinator/retrieveSecretByID
			// 查询 userID 是否存在
			res, err := httpClient.Get("http://control-plane.console.svc.cluster.local:9901/combinator/retrieveSecretByID?user_id=" + userID)
			if err != nil || res.StatusCode != 200 {
				c.AbortWithStatusJSON(500, gin.H{
					"error": "Internal error: failed to retrieve secret",
				})
				return
			}
			var result returntype
			if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
				c.AbortWithStatusJSON(500, gin.H{
					"error": "Internal error: failed to parse secret response",
				})
				return
			}
			gw.userCredentialsLock.Lock()
			gw.userCredentials[userID] = result.SecretKey
			gw.userCredentialsLock.Unlock()
			secretKey = result.SecretKey
		}
		// 使用hmac计算signature
		// 获取body
		body, err := c.GetRawData()
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{
				"error": "Failed to read request body",
			})
			return
		}
		if err := common.VerifyHMACSignature(secretKey, body, signature); err != nil {
			c.AbortWithStatusJSON(401, gin.H{
				"error": "Invalid signature",
			})
			return
		}
		// 将body重新写回请求中，以便后续处理
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		c.Next()
	})
}

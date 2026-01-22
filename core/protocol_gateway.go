package combinator

import (
	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
	rdbModule "jabberwocky238/combinator/core/rdb"
)

type Gateway struct {
	g          *gin.Engine
	rdbGateway *rdbModule.RDBGateway
}

func NewGateway(conf *common.Config) *Gateway {
	r := gin.Default()
	return &Gateway{
		g:          r,
		rdbGateway: rdbModule.NewGateway(r.Group("/rdb"), conf.Rdb),
	}
}

func (gw *Gateway) Start(addr string) error {
	err := gw.rdbGateway.Start()
	if err != nil {
		return err
	}
	return gw.g.Run(addr)
}

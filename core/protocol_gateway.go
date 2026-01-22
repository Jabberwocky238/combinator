package combinator

import (
	"github.com/gin-gonic/gin"

	common "jabberwocky238/combinator/core/common"
	kvModule "jabberwocky238/combinator/core/kv"
	rdbModule "jabberwocky238/combinator/core/rdb"
)

type Gateway struct {
	g          *gin.Engine
	rdbGateway *rdbModule.RDBGateway
	kvGateway  *kvModule.KVGateway
}

func NewGateway(conf *common.Config) *Gateway {
	r := gin.Default()
	return &Gateway{
		g:          r,
		rdbGateway: rdbModule.NewGateway(r.Group("/rdb"), conf.Rdb),
		kvGateway:  kvModule.NewGateway(r.Group("/kv"), conf.Kv),
	}
}

func (gw *Gateway) Start(addr string) error {
	err := gw.rdbGateway.Start()
	if err != nil {
		return err
	}

	err = gw.kvGateway.Start()
	if err != nil {
		return err
	}

	return gw.g.Run(addr)
}

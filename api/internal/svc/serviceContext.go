package svc

import (
	"cjgt/app/bi/cmd/api/internal/config"
	"cjgt/app/bi/cmd/rpc/bi"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	BiRpc bi.Bi
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		BiRpc:  bi.NewBi(zrpc.MustNewClient(c.BiRpcConf)),
	}
}

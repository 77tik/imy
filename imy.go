package main

import (
	"flag"

	"imy/internal/config"
	"imy/internal/dao"
	"imy/internal/handler"
	"imy/internal/svc"
	"imy/pkg/utils"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/imy-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// swagger
	handler.RegisterSwaggerHandlers(server, ctx)

	// ws
	handler.RegisterWsHandlersV2(server, ctx)

	ServerInit(ctx)

	// validate
	httpx.SetValidator(utils.NewValidator())

	logx.Infof("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

func ServerInit(ctx *svc.ServiceContext) {
	logx.MustSetup(ctx.Config.Log)

	// mysql
	dao.SetDefault(ctx.Mysql)

}

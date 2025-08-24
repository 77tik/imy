package handler

import (
	"net/http"

	chat "imy/internal/handler/chat"
	"imy/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

// RegisterWsHandlers registers custom WebSocket routes (non-goctl generated).
func RegisterWsHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/ws",
				Handler: chat.ChatWsHandler(serverCtx),
			},
		},
		rest.WithPrefix("/api/chat"),
	)
}
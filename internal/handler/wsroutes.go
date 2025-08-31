package handler

import (
	"encoding/json"
	"net/http"

	"imy/internal/handler/chat"
	"imy/internal/svc"
	xhttp "imy/pkg/httpx"
	"imy/pkg/websocket"

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

// RegisterWsHandlers registers custom WebSocket routes (non-goctl generated).
func RegisterWsHandlersV2(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/ws",
				Handler: WsHandlerV2(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/v2"),
	)
}

func WsHandlerV2(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cw := &xhttp.CustomResponseWriter{
			ResponseWriter: w,
			Wrote:          false,
		}
		ctx := xhttp.HttpInterceptor(r.Context(), cw, r)
		userIdNumber, _ := ctx.Value("userId").(json.Number)
		userId64, err := userIdNumber.Int64()
		userId := uint32(userId64)

		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 2. 调用 ServeWsWithAuth 升级连接
		websocket.ServeWsWithAuth(serverCtx.WsHub, w, r, userId)
	}
}

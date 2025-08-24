package httpx

import (
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/rest"
	"imy/pkg/jwt"
)

// StartWithInjectJwt 启动时自动注入指定的 JWT 令牌
func StartWithInjectJwt(server *rest.Server, jwt string, inject bool) {
	server.StartWithOpts(func(svr *http.Server) {
		appHandler := svr.Handler
		svr.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if inject && len(r.Header.Get("Authorization")) == 0 {
				r.Header.Set("Authorization", "Bearer "+jwt)
			}
			appHandler.ServeHTTP(w, r)
		})
	})
}

// StartWithInjectUserJwt 启动时自动生成并注入指定的用户
func StartWithInjectUserJwt[T any](server *rest.Server, secret string, userId T, inject bool) {
	token, err := jwt.GenerateToken(jwt.Config{
		SecretKey: []byte(secret),
		Issuer:    "faker",
		ExpiresAt: 365 * 24 * time.Hour,
	}, map[string]any{"UUID": userId, "UserName": "FakeUser"})
	if err != nil {
		panic(err)
	}

	StartWithInjectJwt(server, token, inject)
}

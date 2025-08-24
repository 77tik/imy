package chat

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"imy/internal/svc"
	"imy/pkg/jwt"
)

// ChatWsHandler handles WebSocket upgrade with auth and a minimal read/ping loop.
func ChatWsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// TODO: Restrict origin per configuration. For now, allow all origins for development.
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// helper to parse Authorization: Bearer <token>
	parseBearer := func(h http.Header) (string, error) {
		v := h.Get("Authorization")
		if v == "" {
			return "", errors.New("missing Authorization header")
		}
		parts := strings.SplitN(v, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			return "", errors.New("invalid Authorization header")
		}
		return parts[1], nil
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// 1) auth
		tok, err := parseBearer(r.Header)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		claims, err := jwt.ParseToken(tok, svcCtx.Config.Auth.AccessSecret)
		if err != nil || claims == nil || claims.UUID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		uuid := claims.UUID

		// 2) upgrade
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logx.Errorf("ws upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		// register and ensure unregister on exit
		svcCtx.Ws.Register(uuid, conn)
		defer svcCtx.Ws.Unregister(uuid, conn)

		// Read setup
		conn.SetReadLimit(64 << 10) // 64KB per message
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		// Send a minimal ready event
		type readyPayload struct {
			Op   string                 `json:"op"`
			Data map[string]interface{} `json:"data"`
		}
		_ = svcCtx.Ws.WithConnWrite(conn, func(c *websocket.Conn) error {
			return c.WriteJSON(readyPayload{Op: "ready", Data: map[string]interface{}{"serverTime": time.Now().Unix(), "uuid": uuid}})
		})

		// Ping loop
		done := make(chan struct{})
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer func() {
				ticker.Stop()
				close(done)
			}()
			for {
				select {
				case <-r.Context().Done():
					return
				case <-ticker.C:
					_ = svcCtx.Ws.WithConnWrite(conn, func(c *websocket.Conn) error {
						_ = c.SetWriteDeadline(time.Now().Add(10 * time.Second))
						return c.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(10*time.Second))
					})
				}
			}
		}()

		// Read loop (placeholder)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				// Normal closure or error
				break
			}
		}

		<-done
	}
}
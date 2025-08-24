package chat

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"context"

	"imy/internal/logic/chat"
	"imy/internal/svc"
	"imy/internal/types"

	xhttp "imy/pkg/httpx"
)

func UpdateConversationSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// snapshot raw body to detect field presence
		var rawBody []byte
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			rawBody = b
			r.Body = io.NopCloser(bytes.NewReader(b))
		}

		var req types.UpdateConversationSettingsReq
		if err := xhttp.Parse(r, &req); err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
			return
		}
		cw := &xhttp.CustomResponseWriter{
			ResponseWriter: w,
			Wrote:          false,
		}
		ctx := xhttp.HttpInterceptor(r.Context(), cw, r)

		// detect presence of fields in JSON
		presence := chat.UpdateSettingsPresence{}
		if len(rawBody) > 0 {
			var obj map[string]json.RawMessage
			if err := json.Unmarshal(rawBody, &obj); err == nil {
				if _, ok := obj["alias"]; ok { presence.Alias = true }
				if _, ok := obj["muteUntil"]; ok { presence.MuteUntil = true }
				if _, ok := obj["isPinned"]; ok { presence.IsPinned = true }
			}
		}
		ctx = context.WithValue(ctx, chat.UpdateSettingsPresenceKey{}, presence)

		l := chat.NewUpdateConversationSettingsLogic(ctx, svcCtx)
		err := l.UpdateConversationSettings(&req)
		if err != nil {
			if !cw.Wrote {
				xhttp.JsonBaseResponseCtx(r.Context(), w, err)
			}
		} else {
			if !cw.Wrote {
				xhttp.JsonBaseResponseCtx(r.Context(), w, nil)
			}
		}
	}
}

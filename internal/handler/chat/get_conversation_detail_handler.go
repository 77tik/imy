package chat

import (
	"net/http"

	"imy/internal/logic/chat"
	"imy/internal/svc"
	"imy/internal/types"

	xhttp "imy/pkg/httpx"
)

func GetConversationDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetConversationDetailReq
		if err := xhttp.Parse(r, &req); err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
			return
		}
		cw := &xhttp.CustomResponseWriter{
			ResponseWriter: w,
			Wrote:          false,
		}
		ctx := xhttp.HttpInterceptor(r.Context(), cw, r)

		l := chat.NewGetConversationDetailLogic(ctx, svcCtx)
		resp, err := l.GetConversationDetail(&req)
		if err != nil {
			if !cw.Wrote {
				xhttp.JsonBaseResponseCtx(r.Context(), w, err)
			}
		} else {
			if !cw.Wrote {
				xhttp.JsonBaseResponseCtx(r.Context(), w, resp)
			}
		}
	}
}

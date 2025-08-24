package auth

import (
	"net/http"

	"imy/internal/logic/auth"
	"imy/internal/svc"
	"imy/internal/types"

	xhttp "imy/pkg/httpx"
)

func EmailPasswordLoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.EmailPasswordLoginReq
		if err := xhttp.Parse(r, &req); err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
			return
		}
		cw := &xhttp.CustomResponseWriter{
			ResponseWriter: w,
			Wrote:          false,
		}
		ctx := xhttp.HttpInterceptor(r.Context(), cw, r)

		l := auth.NewEmailPasswordLoginLogic(ctx, svcCtx)
		resp, err := l.EmailPasswordLogin(&req)
		if err != nil {
			if !cw.Wrote {
				// use cw to preserve any headers set in logic
				xhttp.JsonBaseResponseCtx(r.Context(), cw, err)
			}
		} else {
			if !cw.Wrote {
				// use cw to preserve any headers set in logic
				xhttp.JsonBaseResponseCtx(r.Context(), cw, resp)
			}
		}
	}
}

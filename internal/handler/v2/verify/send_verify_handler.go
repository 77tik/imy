package verify

import (
	"net/http"

	"imy/internal/logic/v2/verify"
	"imy/internal/svc"
	"imy/internal/types"

	xhttp "imy/pkg/httpx"
)

func SendVerifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SendVerifyReq
		if err := xhttp.Parse(r, &req); err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
			return
		}
		cw := &xhttp.CustomResponseWriter{
			ResponseWriter: w,
			Wrote:          false,
		}
		ctx := xhttp.HttpInterceptor(r.Context(), cw, r)

		l := verify.NewSendVerifyLogic(ctx, svcCtx)
		err := l.SendVerify(&req)
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

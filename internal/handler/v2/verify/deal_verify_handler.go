package verify

import (
	"net/http"

	"imy/internal/logic/v2/verify"
	"imy/internal/svc"
	"imy/internal/types"

	xhttp "imy/pkg/httpx"
)

func DealVerifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DealVerifyReq
		if err := xhttp.Parse(r, &req); err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
			return
		}
		cw := &xhttp.CustomResponseWriter{
			ResponseWriter: w,
			Wrote:          false,
		}
		ctx := xhttp.HttpInterceptor(r.Context(), cw, r)

		l := verify.NewDealVerifyLogic(ctx, svcCtx)
		err := l.DealVerify(&req)
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

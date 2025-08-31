package user

import (
	"net/http"

	"imy/internal/logic/v2/user"
	"imy/internal/svc"
	"imy/internal/types"

	xhttp "imy/pkg/httpx"
)

func GetAccountByNameHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetAccountByNameReq
		if err := xhttp.Parse(r, &req); err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
			return
		}
		cw := &xhttp.CustomResponseWriter{
			ResponseWriter: w,
			Wrote:          false,
		}
		ctx := xhttp.HttpInterceptor(r.Context(), cw, r)

		l := user.NewGetAccountByNameLogic(ctx, svcCtx)
		resp, err := l.GetAccountByName(&req)
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

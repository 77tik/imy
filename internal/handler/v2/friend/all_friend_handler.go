package friend

import (
	"net/http"

	"imy/internal/logic/v2/friend"
	"imy/internal/svc"

	xhttp "imy/pkg/httpx"
)

func AllFriendHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cw := &xhttp.CustomResponseWriter{
			ResponseWriter: w,
			Wrote:          false,
		}
		ctx := xhttp.HttpInterceptor(r.Context(), cw, r)

		l := friend.NewAllFriendLogic(ctx, svcCtx)
		resp, err := l.AllFriend()
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

package friend

import (
	"net/http"

	"imy/internal/logic/friend"
	"imy/internal/svc"
	"imy/internal/types"

	xhttp "imy/pkg/httpx"
	"github.com/zeromicro/go-zero/core/logx"
)

func AddFriendHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AddFriendReq
		
		// 添加调试日志：打印所有请求头
		logx.Infof("Request headers: %+v", r.Header)
		logx.Infof("UUID header value: %s", r.Header.Get("uuid"))
		
		if err := xhttp.Parse(r, &req); err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
			return
		}
		
		// 添加调试日志：打印解析后的请求结构体
		logx.Infof("Parsed request: UUID=%s, RevId=%s", req.UUID, req.RevId)
		cw := &xhttp.CustomResponseWriter{
			ResponseWriter: w,
			Wrote:          false,
		}
		ctx := xhttp.HttpInterceptor(r.Context(), cw, r)

		l := friend.NewAddFriendLogic(ctx, svcCtx)
		err := l.AddFriend(&req)
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

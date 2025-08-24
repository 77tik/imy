package friend

import (
	"context"
	"errors"
	"strings"

	"imy/internal/dao"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"
	"imy/pkg/httpx"
	"imy/pkg/jwt"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type SearchUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 搜索用户
func NewSearchUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchUserLogic {
	return &SearchUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchUserLogic) SearchUser(req *types.SearchUserReq) (resp *types.SearchUserResp, err error) {
	// 先用email去用户表查是否存在该用户
	user, err := dao.User.WithContext(l.ctx).Where(dao.User.Email.Eq(req.Email)).Take()
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrAuthUserNotFund.WithError(err)
	}
	if err != nil {
		return nil, errcode.ErrDataQueryFail.WithError(err)
	}

	// 不能搜到自己：从请求头解析 token，拿到当前用户 UUID，与目标用户对比
	if r, ok := httpx.GetRequest(l.ctx); ok {
		auth := r.Header.Get("Authorization")
		token := ""
		if auth != "" {
			if strings.HasPrefix(auth, "Bearer ") {
				token = strings.TrimSpace(auth[len("Bearer "):])
			} else {
				token = auth
			}
		}
		if token == "" {
			token = r.Header.Get("token")
		}
		if token != "" {
			if claims, perr := jwt.ParseToken(token, l.svcCtx.Config.Auth.AccessSecret); perr == nil && claims != nil {
				if claims.UUID == user.UUID {
					// 与当前用户相同，则视为未找到
					return nil, errcode.ErrAuthUserNotFund
				}
			}
		}
	}

	// 如果存在就返回该用户UUID
	return &types.SearchUserResp{
		RevId: user.UUID,
	}, nil
}

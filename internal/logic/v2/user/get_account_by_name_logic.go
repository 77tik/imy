package user

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"imy/internal/dao"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAccountByNameLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 根据用户名得到账号
func NewGetAccountByNameLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAccountByNameLogic {
	return &GetAccountByNameLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAccountByNameLogic) GetAccountByName(req *types.GetAccountByNameReq) (resp *types.GetAccountByNameResp, err error) {
	auth, err := dao.Auth.GetByNickname(l.ctx, req.Nickname)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAuthUserNotFund.WithError(err)
		}
		return nil, errcode.ErrDataQueryFail.WithError(err)
	}

	return &types.GetAccountByNameResp{Account: auth.Account}, nil
}

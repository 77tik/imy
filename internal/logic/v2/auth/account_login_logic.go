package auth

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"imy/internal/dao"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"
	"imy/pkg/jwt"
	"imy/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AccountLoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 账号登陆
func NewAccountLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AccountLoginLogic {
	return &AccountLoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AccountLoginLogic) AccountLogin(req *types.AccountLoginReq) (resp *types.AccountLoginResp, err error) {
	if req.Account == "" || req.Password == "" {
		return nil, errcode.ErrInvalidParam.WithError(err)
	}
	auth, err := dao.Auth.GetByAccount(l.ctx, req.Account)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAuthUserNotFund.WithError(err)
		}
		return nil, errcode.ErrDataQueryFail.WithError(err)
	}

	err = utils.PwdVerify(req.Password, auth.Password)
	if err != nil {
		return nil, errcode.ErrInvalidParam.WithError(err)
	}

	token, err := jwt.GenerateToken(jwt.Config{
		SecretKey: []byte(l.svcCtx.Config.Auth.AccessSecret),
		Issuer:    "7878",
		ExpiresAt: time.Duration(l.svcCtx.Config.Auth.AccessExpire) * time.Second,
	}, map[string]interface{}{
		"userId":   auth.ID,
		"userName": auth.NickName,
	})
	if err != nil {
		return nil, errcode.ErrAuthTokenCreateFailed.WithError(err)
	}

	return &types.AccountLoginResp{Token: token}, nil
}

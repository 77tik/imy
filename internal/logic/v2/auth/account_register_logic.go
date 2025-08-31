package auth

import (
	"context"

	"imy/internal/dao"
	"imy/internal/dao/model"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"
	"imy/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AccountRegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 账号注册
func NewAccountRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AccountRegisterLogic {
	return &AccountRegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AccountRegisterLogic) AccountRegister(req *types.AccountRegisterReq) (resp *types.AccountRegisterResp, err error) {
	pwdHash, err := utils.PwdGenerate(req.Password)
	if err != nil {
		return nil, errcode.ErrPasswordGenerate.WithError(err)
	}

	account := l.svcCtx.Snow.Generate().String()
	err = dao.Auth.WithContext(l.ctx).Create(&model.Auth{
		Account:  account,
		NickName: req.Nickname,
		Password: pwdHash,
	})
	if err != nil {
		return nil, errcode.ErrDataCreateFail.WithError(err)
	}

	return &types.AccountRegisterResp{
		Account: account,
	}, nil
}

package verify

import (
	"context"
	"encoding/json"

	"imy/internal/dao"
	"imy/internal/dao/model"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"

	"github.com/samber/lo"
	"github.com/zeromicro/go-zero/core/logx"
)

type AllVerifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取验证消息(所有)
func NewAllVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AllVerifyLogic {
	return &AllVerifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AllVerifyLogic) AllVerify() (resp *types.AllVerifyResp, err error) {
	userIdNumber, ok := l.ctx.Value("userId").(json.Number)
	if !ok {
		return nil, errcode.ErrAuthTokenUseless.WithError(err)
	}
	userId64, err := userIdNumber.Int64()
	userId := uint32(userId64)

	allList := []*model.Verify{}
	sendlist, err := dao.Verify.GetListBySendId(l.ctx, userId)
	if err != nil {
		return nil, errcode.ErrDataQueryFail.WithError(err)
	}
	revlist, err := dao.Verify.GetListByRevId(l.ctx, userId)
	if err != nil {
		return nil, errcode.ErrDataQueryFail.WithError(err)
	}

	allList = append(allList, append(sendlist, revlist...)...)

	return &types.AllVerifyResp{
		Infos: lo.Map(allList, func(item *model.Verify, _ int) types.VerifyInfo {
			return item.DTO()
		}),
	}, nil
}

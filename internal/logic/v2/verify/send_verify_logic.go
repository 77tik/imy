package verify

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
	"imy/internal/dao"
	"imy/internal/dao/model"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendVerifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 发送验证
func NewSendVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendVerifyLogic {
	return &SendVerifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendVerifyLogic) SendVerify(req *types.SendVerifyReq) error {
	ob, err := dao.Auth.GetByAccount(l.ctx, req.Account)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrAuthUserNotFund.WithError(err)
		}
		return errcode.ErrDataQueryFail.WithError(err)
	}
	userIdNumber, ok := l.ctx.Value("userId").(json.Number)
	if !ok {
		return errcode.ErrAuthTokenUseless.WithError(err)
	}
	userId64, err := userIdNumber.Int64()
	userId := uint32(userId64)

	// 查一下是不是已经是好友了
	_, total, err := dao.FriendV2.List(l.ctx, &dao.ListFriendV2Params{
		SendId: userId,
		RevId:  ob.ID,
	})
	if total != 0 {
		return errcode.ErrFriendAlreadyExist.WithError(err)
	}

	// 查一下该验证以前是不是发过
	_, ver_total, err := dao.Verify.List(l.ctx, &dao.ListVerifyParams{
		SendId: userId,
		RevId:  ob.ID,
	})
	if ver_total != 0 {
		return errcode.ErrVerifyExist.WithError(err)
	}

	// 发送好友验证
	err = dao.Verify.WithContext(l.ctx).Create(&model.Verify{
		SendID: userId,
		RevID:  ob.ID,
	})
	if err != nil {
		return errcode.ErrDataCreateFail.WithError(err)
	}
	// TODO:websocket通知对象
	l.svcCtx.WsHub.SendToUserFrom(userId, ob.ID, "收到好友请求")
	return nil
}

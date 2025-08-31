package verify

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"imy/internal/dao"
	"imy/internal/dao/model"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DealVerifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 处理验证
func NewDealVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DealVerifyLogic {
	return &DealVerifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DealVerifyLogic) DealVerify(req *types.DealVerifyReq) error {
	// 查询好友验证是否存在？
	verify, err := dao.Verify.Get(l.ctx, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrVerifyNotFound.WithError(err)
		}
		return errcode.ErrDataQueryFail.WithError(err)
	}

	verify.Status = req.Status
	err = dao.Verify.Update(l.ctx, verify)
	if err != nil {
		return errcode.ErrDataModifyFail.WithError(err)
	}

	// TODO:websocket 通知对方 我已经同意了申请
	l.svcCtx.WsHub.SendToUserFrom(verify.SendID, verify.RevID, "对方已经同意了你的验证")

	err = dao.FriendV2.WithContext(l.ctx).Create(&model.FriendV2{
		SendID: verify.SendID,
		RevID:  verify.RevID,
	})
	if err != nil {
		return errcode.ErrDataCreateFail.WithError(err)
	}
	return nil
}

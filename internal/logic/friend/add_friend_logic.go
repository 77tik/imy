package friend

import (
	"context"
	"errors"

	"imy/internal/dao"
	"imy/internal/dao/model"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type AddFriendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 添加好友
func NewAddFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddFriendLogic {
	return &AddFriendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddFriendLogic) AddFriend(req *types.AddFriendReq) error {
	// 添加调试日志
	l.Logger.Infof("AddFriend request: UUID=%s, RevId=%s", req.UUID, req.RevId)
	
	// 校验参数
	if req.UUID == "" || req.RevId == "" {
		l.Logger.Errorf("Invalid param: UUID=%s, RevId=%s", req.UUID, req.RevId)
		return errcode.ErrInvalidParam
	}
	if req.UUID == req.RevId {
		l.Logger.Errorf("UUID equals RevId: %s", req.UUID)
		return errcode.ErrInvalidParam
	}

	// 新增校验：如果已经是好友，则不能再次添加
	_, err := dao.Friend.WithContext(l.ctx).
		Where(dao.Friend.SendUUID.Eq(req.UUID), dao.Friend.RevUUID.Eq(req.RevId)).
		Or(dao.Friend.SendUUID.Eq(req.RevId), dao.Friend.RevUUID.Eq(req.UUID)).
		Take()
	if err == nil {
		return errcode.ErrFriendAlreadyExist
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return errcode.ErrDataQueryFail.WithError(err)
	}

	// 不能重复存在未处理的验证请求（发送方）
	_, err = dao.FriendVerify.WithContext(l.ctx).
		Where(dao.FriendVerify.SendUUID.Eq(req.UUID), dao.FriendVerify.RevUUID.Eq(req.RevId), dao.FriendVerify.RevStatus.Eq(1)).
		Take()
	if err == nil {
		// 已存在待处理记录，幂等返回成功
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return errcode.ErrDataQueryFail.WithError(err)
	}

	// 创建好友验证记录
	fv := &model.FriendVerify{
		SendUUID:   req.UUID,
		RevUUID:    req.RevId,
		SendStatus: 1, // 待处理
		RevStatus:  1, // 待处理
		Source:     1, // 默认来源：搜索
	}
	if err := dao.FriendVerify.WithContext(l.ctx).Create(fv); err != nil {
		return errcode.ErrDataCreateFail.WithError(err)
	}
	return nil
}

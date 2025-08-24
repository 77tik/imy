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

type ValidFriendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 处理好友验证
func NewValidFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ValidFriendLogic {
	return &ValidFriendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ValidFriendLogic) ValidFriend(req *types.ValidFriendReq) error {
	// 1. 从好友验证表中查询该条验证记录
	if req.UUID == "" || req.VerifyId == 0 {
		return errcode.ErrInvalidParam
	}

	verify, err := dao.FriendVerify.WithContext(l.ctx).Where(dao.FriendVerify.ID.Eq(req.VerifyId)).Take()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrDataQueryFail.WithError(err)
		}
		return errcode.ErrDataQueryFail.WithError(err)
	}

	// 2. 校验操作人是否是该条验证的接收者
	if verify.RevUUID != req.UUID {
		return errcode.ErrInvalidParam
	}

	switch req.Status {
	case 1: // 同意
		// 在事务中：向好友表添加数据，并将验证条目接收者状态改为2（已同意）
		return dao.Q.Transaction(func(tx *dao.Query) error {
			// 若不存在相同关系则插入
			_, err := tx.Friend.WithContext(l.ctx).
				Where(dao.Friend.SendUUID.Eq(verify.SendUUID), dao.Friend.RevUUID.Eq(verify.RevUUID)).
				Take()
			if err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return errcode.ErrDataQueryFail.WithError(err)
				}
				// 不存在则创建，默认备注使用对方的昵称
				var senderNick, receiverNick string
				if su, e1 := tx.User.WithContext(l.ctx).
					Select(dao.User.NickName).
					Where(dao.User.UUID.Eq(verify.SendUUID)).
					Take(); e1 == nil {
					senderNick = su.NickName
				} else if !errors.Is(e1, gorm.ErrRecordNotFound) {
					return errcode.ErrDataQueryFail.WithError(e1)
				}
				if ru, e2 := tx.User.WithContext(l.ctx).
					Select(dao.User.NickName).
					Where(dao.User.UUID.Eq(verify.RevUUID)).
					Take(); e2 == nil {
					receiverNick = ru.NickName
				} else if !errors.Is(e2, gorm.ErrRecordNotFound) {
					return errcode.ErrDataQueryFail.WithError(e2)
				}

				if err := tx.Friend.WithContext(l.ctx).Create(&model.Friend{
					SendUUID:   verify.SendUUID,
					RevUUID:    verify.RevUUID,
					// 发起方对对方的备注=接收者昵称；接收方对对方的备注=发起者昵称
					SendNotice: receiverNick,
					RevNotice:  senderNick,
					Source:     verify.Source,
				}); err != nil {
					return errcode.ErrDataCreateFail.WithError(err)
				}
			}

			// 更新验证记录接收者状态为2
			upd := &model.FriendVerify{ID: verify.ID, RevStatus: 2}
			if err := (&tx.FriendVerify).Update(l.ctx, upd, "RevStatus"); err != nil {
				return errcode.ErrDataModifyFail.WithError(err)
			}
			return nil
		})
	case 2: // 拒绝
		upd := &model.FriendVerify{ID: verify.ID, RevStatus: 3}
		if err := dao.FriendVerify.Update(l.ctx, upd, "RevStatus"); err != nil {
			return errcode.ErrDataModifyFail.WithError(err)
		}
		return nil
	default:
		return errcode.ErrInvalidParam
	}
}

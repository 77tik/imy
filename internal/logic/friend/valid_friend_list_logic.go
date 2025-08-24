package friend

import (
	"context"

	"imy/internal/dao"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ValidFriendListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取好友验证表
func NewValidFriendListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ValidFriendListLogic {
	return &ValidFriendListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ValidFriendListLogic) ValidFriendList(req *types.ValidFriendListReq) (resp *types.ValidFriendListResp, err error) {
	// 获取当前用户作为接收方的好友验证条目
	if req.UUID == "" {
		return nil, errcode.ErrInvalidParam
	}

	list, err := dao.FriendVerify.WithContext(l.ctx).
		Where(dao.FriendVerify.RevUUID.Eq(req.UUID)).
		Find()
	if err != nil {
		return nil, errcode.ErrDataQueryFail.WithError(err)
	}

	items := make([]types.ValidFriendInfo, 0, len(list))
	for _, v := range list {
		items = append(items, types.ValidFriendInfo{
			Id:        v.ID,
			RevId:     v.SendUUID, // 返回发送方的UUID，这样接收方可以看到是谁发送的请求
			RevStatus: uint32(v.RevStatus),
		})
	}
	return &types.ValidFriendListResp{Valids: items}, nil
}

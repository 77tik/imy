package friend

import (
	"context"

	"imy/internal/dao"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFriendListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取好友列表
func NewGetFriendListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFriendListLogic {
	return &GetFriendListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetFriendListLogic) GetFriendList(req *types.GetFriendListReq) (resp *types.GetFriendListResp, err error) {
	// 参数校验
	if req.UUID == "" {
		return nil, errcode.ErrInvalidParam
	}

	// 根据 uuid 查询好友表，查出自己的好友及其备注
	list, err := dao.Friend.WithContext(l.ctx).
		Where(dao.Friend.SendUUID.Eq(req.UUID)).
		Or(dao.Friend.RevUUID.Eq(req.UUID)).
		Find()
	if err != nil {
		return nil, errcode.ErrDataQueryFail.WithError(err)
	}

	friends := make([]types.FriendInfo, 0, len(list))
	for _, f := range list {
		if f.SendUUID == req.UUID {
			friends = append(friends, types.FriendInfo{
				UUID:   f.RevUUID,
				Notice: f.SendNotice,
			})
			continue
		}
		// 否则当前用户是接收者
		friends = append(friends, types.FriendInfo{
			UUID:   f.SendUUID,
			Notice: f.RevNotice,
		})
	}

	return &types.GetFriendListResp{Friends: friends}, nil
}

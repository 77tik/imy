package friend

import (
	"context"
	"encoding/json"

	"github.com/samber/lo"
	"imy/internal/dao"
	"imy/internal/dao/model"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AllFriendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取好友列表
func NewAllFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AllFriendLogic {
	return &AllFriendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AllFriendLogic) AllFriend() (resp *types.AllFriendResp, err error) {
	userIdNumber, ok := l.ctx.Value("userId").(json.Number)
	if !ok {
		return nil, errcode.ErrAuthTokenUseless.WithError(err)
	}
	userId64, err := userIdNumber.Int64()
	userId := uint32(userId64)
	list, _, err := dao.FriendV2.List(l.ctx, &dao.ListFriendV2Params{
		SendId: userId,
	})
	return &types.AllFriendResp{
		Infos: lo.Map(list, func(item *model.FriendV2, _ int) types.FriendV2Info {
			auth, _ := dao.Auth.Get(l.ctx, item.RevID)
			return types.FriendV2Info{
				Account:  auth.Account,
				Nickname: auth.NickName,
			}
		}),
	}, nil
}

package chat

import (
	"context"

	"imy/internal/dao"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUnreadCountsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取未读计数
func NewGetUnreadCountsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUnreadCountsLogic {
	return &GetUnreadCountsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUnreadCountsLogic) GetUnreadCounts(req *types.GetUnreadCountsReq) (resp *types.GetUnreadCountsResp, err error) {
	if req.UUID == "" {
		return nil, errcode.ErrInvalidParam
	}

	// 查询该用户加入的所有会话成员记录
	members, e := dao.ChatConversationMember.WithContext(l.ctx).
		Where(dao.ChatConversationMember.UserUUID.Eq(req.UUID)).
		Find()
	if e != nil {
		return nil, errcode.ErrDataQueryFail.WithError(e)
	}

	items := make([]types.UnreadItem, 0, len(members))
	for _, m := range members {
		// 统计该会话中大于 last_read_message_id 的消息数，排除自己发送的消息
		cnt, e := dao.ChatMessage.WithContext(l.ctx).
			Where(
				dao.ChatMessage.ConversationID.Eq(m.ConversationID),
				dao.ChatMessage.ID.Gt(m.LastReadMessageID),
				dao.ChatMessage.SendUUID.Neq(req.UUID),
			).
			Count()
		if e != nil {
			return nil, errcode.ErrDataQueryFail.WithError(e)
		}
		items = append(items, types.UnreadItem{
			ConversationId: m.ConversationID,
			Unread:         uint32(cnt),
		})
	}

	return &types.GetUnreadCountsResp{Items: items}, nil
}

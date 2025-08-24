package chat

import (
	"context"

	"imy/internal/dao"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetConversationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取我的会话列表
func NewGetConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationsLogic {
	return &GetConversationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetConversationsLogic) GetConversations(req *types.GetConversationsReq) (resp *types.GetConversationsResp, err error) {
	if req.UUID == "" {
		return nil, errcode.ErrInvalidParam
	}
	pageSize := req.PageSize
	pageIndex := req.PageIndex
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
	if pageIndex <= 0 {
		pageIndex = 1
	}
	offset := (pageIndex - 1) * pageSize

	// 通过 join 成员表筛选出用户所在会话，按 last_message_id 倒序分页
	convs, e := dao.ChatConversation.WithContext(l.ctx).
		Join(dao.ChatConversationMember, dao.ChatConversationMember.ConversationID.EqCol(dao.ChatConversation.ID)).
		Where(dao.ChatConversationMember.UserUUID.Eq(req.UUID)).
		Order(dao.ChatConversation.LastMessageID.Desc()).
		Offset(offset).Limit(pageSize).
		Find()
	if e != nil {
		return nil, errcode.ErrDataQueryFail.WithError(e)
	}

	list := make([]types.ConversationInfo, 0, len(convs))
	for _, c := range convs {
		list = append(list, types.ConversationInfo{
			ConversationId: c.ID,
			Type:           uint32(c.Type),
			PrivateKey:     c.PrivateKey,
			Name:           c.Name,
			MemberCount:    c.MemberCount,
			LastMessageId:  c.LastMessageID,
			Avatar:         c.Avatar,
			Extra:          c.Extra,
		})
	}

	return &types.GetConversationsResp{Conversations: list}, nil
}

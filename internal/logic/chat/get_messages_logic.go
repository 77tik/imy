package chat

import (
	"context"
	"errors"
	"strings"
	"time"

	"imy/internal/dao"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type GetMessagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 拉取历史消息
func NewGetMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMessagesLogic {
	return &GetMessagesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMessagesLogic) GetMessages(req *types.GetMessagesReq) (resp *types.GetMessagesResp, err error) {
	// 1) 参数校验
	if req.UUID == "" || req.ConversationId == 0 {
		return nil, errcode.ErrInvalidParam
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	// 2) 校验是否会话成员
	if _, e := dao.ChatConversationMember.WithContext(l.ctx).
		Where(dao.ChatConversationMember.ConversationID.Eq(req.ConversationId), dao.ChatConversationMember.UserUUID.Eq(req.UUID)).
		Take(); e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAuthSession
		}
		return nil, errcode.ErrDataQueryFail.WithError(e)
	}

	// 3) 组装查询
	q := dao.ChatMessage.WithContext(l.ctx).Where(dao.ChatMessage.ConversationID.Eq(req.ConversationId))
	orderAsc := false
	if req.AfterId > 0 {
		q = q.Where(dao.ChatMessage.ID.Gt(req.AfterId)).Order(dao.ChatMessage.ID.Asc())
		orderAsc = true
	} else if req.BeforeId > 0 {
		q = q.Where(dao.ChatMessage.ID.Lt(req.BeforeId)).Order(dao.ChatMessage.ID.Desc())
		orderAsc = false
	} else {
		q = q.Order(dao.ChatMessage.ID.Desc())
		orderAsc = false
	}
	q = q.Limit(limit)

	list, e := q.Find()
	if e != nil {
		return nil, errcode.ErrDataQueryFail.WithError(e)
	}

	// 4) 按需翻转为升序输出
	if !orderAsc {
		for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
			list[i], list[j] = list[j], list[i]
		}
	}

	// 5) 映射为响应
	msgs := make([]types.MessageInfo, 0, len(list))
	for _, m := range list {
		var mentioned []string
		if m.MentionedUuids != "" {
			mentioned = strings.Split(m.MentionedUuids, ",")
		}
		msgs = append(msgs, types.MessageInfo{
			Id:               m.ID,
			ConversationId:   m.ConversationID,
			SendUuid:         m.SendUUID,
			MsgType:          uint32(m.MsgType),
			Content:          m.Content,
			ContentExtra:     m.ContentExtra,
			ReplyToMessageId: m.ReplyToMessageID,
			MentionedUuids:   mentioned,
			IsSystem:         ternary(m.IsSystem, uint32(1), uint32(0)),
			IsRevoked:        ternary(m.IsRevoked, uint32(1), uint32(0)),
			CreatedAt:        m.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	return &types.GetMessagesResp{Messages: msgs}, nil
}

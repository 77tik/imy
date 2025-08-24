package chat

import (
	"context"
	"errors"
	"strings"
	"time"

	"imy/internal/dao"
	"imy/internal/dao/model"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type SendMessageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 发送消息
func NewSendMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendMessageLogic {
	return &SendMessageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendMessageLogic) SendMessage(req *types.SendMessageReq) (resp *types.SendMessageResp, err error) {
	// 1) 参数校验
	if req.UUID == "" || req.ConversationId == 0 || req.ClientMsgId == "" || req.MsgType == 0 {
		return nil, errcode.ErrInvalidParam
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

	// 3) 幂等：检查是否已存在相同 clientMsgId 的消息
	exist, e := dao.ChatMessage.WithContext(l.ctx).
		Where(
			dao.ChatMessage.ConversationID.Eq(req.ConversationId),
			dao.ChatMessage.SendUUID.Eq(req.UUID),
			dao.ChatMessage.ClientMsgID.Eq(req.ClientMsgId),
		).
		Take()
	if e == nil {
		// 已存在，直接返回
		createdAt := exist.CreatedAt.UTC().Format(time.RFC3339)
		return &types.SendMessageResp{
			ServerMsgId: exist.ID,
			ClientMsgId: exist.ClientMsgID,
			CreatedAt:   createdAt,
		}, nil
	}
	if !errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrDataQueryFail.WithError(e)
	}

	// 4) 写入消息
	mentionedStr := ""
	if len(req.MentionedUuids) > 0 {
		mentionedStr = strings.Join(req.MentionedUuids, ",")
	}
	isSystem := req.MsgType == 6
	msg := &model.ChatMessage{
		ConversationID:   req.ConversationId,
		SendUUID:         req.UUID,
		ClientMsgID:      req.ClientMsgId,
		MsgType:          int8(req.MsgType),
		Content:          req.Content,
		ContentExtra:     req.ContentExtra,
		ReplyToMessageID: req.ReplyToMessageId,
		MentionedUuids:   mentionedStr,
		IsSystem:         isSystem,
		IsRevoked:        false,
	}
	if e := dao.ChatMessage.WithContext(l.ctx).Create(msg); e != nil {
		return nil, errcode.ErrDataCreateFail.WithError(e)
	}

	// 4.1) 更新会话的最后消息ID（忽略错误，不阻塞发送流程）
	_ = dao.ChatConversation.Update(l.ctx, &model.ChatConversation{
		ID:            req.ConversationId,
		LastMessageID: msg.ID,
	}, "LastMessageID")

	// 5) 构造响应
	createdAt := msg.CreatedAt.UTC().Format(time.RFC3339)
	resp = &types.SendMessageResp{
		ServerMsgId: msg.ID,
		ClientMsgId: msg.ClientMsgID,
		CreatedAt:   createdAt,
	}

	// 6) 广播 WS 事件给该会话的所有成员
	go func(m *model.ChatMessage) {
		defer func() { recover() }()
		members, e := dao.ChatConversationMember.WithContext(l.ctx).
			Where(dao.ChatConversationMember.ConversationID.Eq(req.ConversationId)).
			Find()
		if e != nil {
			logx.Errorf("ws broadcast list members failed: %v", e)
			return
		}
		var mentioned []string
		if m.MentionedUuids != "" {
			mentioned = strings.Split(m.MentionedUuids, ",")
		}
		payloadNew := struct {
			Op   string            `json:"op"`
			Data types.MessageInfo `json:"data"`
		}{
			Op: "message_new",
			Data: types.MessageInfo{
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
			},
		}
		for _, mem := range members {
			// 推送新消息
			l.svcCtx.Ws.SendJSON(mem.UserUUID, payloadNew)

			// 计算并推送未读变更：统计 > last_read_message_id 且 发送者 != 自己 的消息数
			cnt, errCnt := dao.ChatMessage.WithContext(l.ctx).
				Where(
					dao.ChatMessage.ConversationID.Eq(m.ConversationID),
					dao.ChatMessage.ID.Gt(mem.LastReadMessageID),
					dao.ChatMessage.SendUUID.Neq(mem.UserUUID),
				).
				Count()
			if errCnt != nil {
				logx.Errorf("ws broadcast unread count failed: %v", errCnt)
				continue
			}
			// 更新缓存未读数（忽略错误）
			mem.UnreadCount = uint32(cnt)
			_ = dao.ChatConversationMember.Update(l.ctx, mem, "UnreadCount")

			payloadUnread := struct {
				Op   string           `json:"op"`
				Data types.UnreadItem `json:"data"`
			}{
				Op: "unread_count_change",
				Data: types.UnreadItem{
					ConversationId: mem.ConversationID,
					Unread:         uint32(cnt),
				},
			}
			l.svcCtx.Ws.SendJSON(mem.UserUUID, payloadUnread)
		}
	}(msg)

	return resp, nil
}

// ternary is a tiny helper to convert bool to uint32(1/0)
func ternary(cond bool, a, b uint32) uint32 {
	if cond {
		return a
	}
	return b
}

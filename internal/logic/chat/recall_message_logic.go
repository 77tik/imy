package chat

import (
	"context"
	"errors"
	"time"

	"imy/internal/dao"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type RecallMessageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 撤回消息
func NewRecallMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecallMessageLogic {
	return &RecallMessageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RecallMessageLogic) RecallMessage(req *types.RecallMessageReq) error {
	// 1) 参数校验
	if req.UUID == "" || req.ConversationId == 0 || req.MessageId == 0 {
		return errcode.ErrInvalidParam
	}

	// 2) 校验操作者是否在该会话内
	if _, e := dao.ChatConversationMember.WithContext(l.ctx).
		Where(
			dao.ChatConversationMember.ConversationID.Eq(req.ConversationId),
			dao.ChatConversationMember.UserUUID.Eq(req.UUID),
		).
		Take(); e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return errcode.ErrAuthSession
		}
		return errcode.ErrDataQueryFail.WithError(e)
	}

	// 3) 读取消息并校验归属与权限
	msg, e := dao.ChatMessage.WithContext(l.ctx).
		Where(
			dao.ChatMessage.ID.Eq(req.MessageId),
			dao.ChatMessage.ConversationID.Eq(req.ConversationId),
		).
		Take()
	if e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return errcode.ErrInvalidParam
		}
		return errcode.ErrDataQueryFail.WithError(e)
	}

	// 允许撤回的条件：
	// - 发送者本人可撤回自己的消息
	// - 或者管理员/群主可撤回（当前模型有 Role 字段：1 普通，2 管理员）。
	isOwner := msg.SendUUID == req.UUID
	if !isOwner {
		// 检查是否管理员
		mem, e := dao.ChatConversationMember.WithContext(l.ctx).
			Where(
				dao.ChatConversationMember.ConversationID.Eq(req.ConversationId),
				dao.ChatConversationMember.UserUUID.Eq(req.UUID),
			).
			Take()
		if e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return errcode.ErrAuthSession
			}
			return errcode.ErrDataQueryFail.WithError(e)
		}
		if mem.Role < 2 { // 1:普通成员, 2:管理员
			return errcode.ErrAuth // 无权限
		}
	}

	// 4) 更新消息为撤回状态
	now := time.Now()
	msg.IsRevoked = true
	msg.RevokedAt = &now
	if e := dao.ChatMessage.Update(l.ctx, msg, "IsRevoked", "RevokedAt"); e != nil {
		return errcode.ErrDataModifyFail.WithError(e)
	}

	// 5) 广播消息撤回事件给会话内所有成员
	go func(conversationID uint32, messageID uint64, operator string) {
		defer func() { recover() }()
		members, err := dao.ChatConversationMember.WithContext(l.ctx).
			Where(dao.ChatConversationMember.ConversationID.Eq(conversationID)).
			Find()
		if err != nil {
			logx.Errorf("ws broadcast recall failed: %v", err)
			return
		}
		payload := struct {
			Op   string `json:"op"`
			Data struct {
				ConversationId uint32 `json:"conversationId"`
				MessageId      uint64 `json:"messageId"`
				OperatorUuid   string `json:"operatorUuid"`
				RevokedAt      string `json:"revokedAt"`
			} `json:"data"`
		}{Op: "message_recalled"}
		payload.Data.ConversationId = conversationID
		payload.Data.MessageId = messageID
		payload.Data.OperatorUuid = operator
		payload.Data.RevokedAt = now.UTC().Format(time.RFC3339)
		for _, m := range members {
			l.svcCtx.Ws.SendJSON(m.UserUUID, payload)
		}
	}(req.ConversationId, req.MessageId, req.UUID)

	return nil
}

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

type ReadMessagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 上报已读进度
func NewReadMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadMessagesLogic {
	return &ReadMessagesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReadMessagesLogic) ReadMessages(req *types.ReadMessagesReq) (resp *types.ReadMessagesResp, err error) {
	// 参数校验
	if req.UUID == "" || req.ConversationId == 0 {
		return nil, errcode.ErrInvalidParam
	}

	// 会话成员校验并获取成员记录
	mem, e := dao.ChatConversationMember.WithContext(l.ctx).
		Where(
			dao.ChatConversationMember.ConversationID.Eq(req.ConversationId),
			dao.ChatConversationMember.UserUUID.Eq(req.UUID),
		).
		Take()
	if e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAuthSession
		}
		return nil, errcode.ErrDataQueryFail.WithError(e)
	}

	// 如果不需要前进，则直接返回当前 last read
	if req.UpToMessageId == 0 || req.UpToMessageId <= mem.LastReadMessageID {
		return &types.ReadMessagesResp{LastReadMessageId: mem.LastReadMessageID}, nil
	}

	// 校验 upToMessageId 属于该会话
	if _, e := dao.ChatMessage.WithContext(l.ctx).
		Where(
			dao.ChatMessage.ID.Eq(req.UpToMessageId),
			dao.ChatMessage.ConversationID.Eq(req.ConversationId),
		).
		Take(); e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrInvalidParam
		}
		return nil, errcode.ErrDataQueryFail.WithError(e)
	}

	// 计算新的 last read
	newLastRead := req.UpToMessageId

	// 计算最新未读数（基于新的 last read），排除自己发送的消息
	cnt, e := dao.ChatMessage.WithContext(l.ctx).
		Where(
			dao.ChatMessage.ConversationID.Eq(req.ConversationId),
			dao.ChatMessage.ID.Gt(newLastRead),
			dao.ChatMessage.SendUUID.Neq(req.UUID),
		).
		Count()
	if e != nil {
		return nil, errcode.ErrDataQueryFail.WithError(e)
	}

	// 更新成员的已读进度与未读缓存
	mem.LastReadMessageID = newLastRead
	mem.LastReadAt = time.Now()
	mem.UnreadCount = uint32(cnt)
	if e := dao.ChatConversationMember.Update(l.ctx, mem, "LastReadMessageID", "LastReadAt", "UnreadCount"); e != nil {
		return nil, errcode.ErrDataModifyFail.WithError(e)
	}

	// 异步推送 WS：向所有成员广播 message_read；向读者本人推送 unread_count_change
	go func(conversationID uint32, reader string, lastReadID uint64, unread uint32) {
		defer func() { recover() }()
		// 广播 message_read
		members, err := dao.ChatConversationMember.WithContext(l.ctx).
			Where(dao.ChatConversationMember.ConversationID.Eq(conversationID)).
			Find()
		if err == nil {
			payloadRead := struct {
				Op   string `json:"op"`
				Data struct {
					ConversationId    uint32 `json:"conversationId"`
					ReaderUuid        string `json:"readerUuid"`
					LastReadMessageId uint64 `json:"lastReadMessageId"`
					ReadAt            string `json:"readAt"`
				} `json:"data"`
			}{
				Op: "message_read",
			}
			payloadRead.Data.ConversationId = conversationID
			payloadRead.Data.ReaderUuid = reader
			payloadRead.Data.LastReadMessageId = lastReadID
			payloadRead.Data.ReadAt = time.Now().UTC().Format(time.RFC3339)
			for _, m := range members {
				l.svcCtx.Ws.SendJSON(m.UserUUID, payloadRead)
			}
		}
		// 向读者推送自己的未读变更
		payloadUnread := struct {
			Op   string           `json:"op"`
			Data types.UnreadItem `json:"data"`
		}{
			Op: "unread_count_change",
			Data: types.UnreadItem{ConversationId: conversationID, Unread: unread},
		}
		l.svcCtx.Ws.SendJSON(reader, payloadUnread)
	}(req.ConversationId, req.UUID, newLastRead, uint32(cnt))

	return &types.ReadMessagesResp{LastReadMessageId: newLastRead}, nil
}

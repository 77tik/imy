package chat

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

type RemoveMemberLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 群聊移除成员/退群
func NewRemoveMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveMemberLogic {
	return &RemoveMemberLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveMemberLogic) RemoveMember(req *types.RemoveMemberReq) error {
	// 校验
	if req.UUID == "" || req.ConversationId == 0 || req.RemoveUUID == "" {
		return errcode.ErrInvalidParam
	}

	// 校验操作者是成员
	if _, e := dao.ChatConversationMember.WithContext(l.ctx).
		Where(dao.ChatConversationMember.ConversationID.Eq(req.ConversationId), dao.ChatConversationMember.UserUUID.Eq(req.UUID)).
		Take(); e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return errcode.ErrAuthSession
		}
		return errcode.ErrDataQueryFail.WithError(e)
	}

	// 被移除者是否在群内
	mem, e := dao.ChatConversationMember.WithContext(l.ctx).
		Where(dao.ChatConversationMember.ConversationID.Eq(req.ConversationId), dao.ChatConversationMember.UserUUID.Eq(req.RemoveUUID)).
		Take()
	if e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return errcode.ErrInvalidParam
		}
		return errcode.ErrDataQueryFail.WithError(e)
	}

	// 删除成员
	if e := dao.ChatConversationMember.DeleteByID(l.ctx, mem.ID); e != nil {
		return errcode.ErrDataModifyFail.WithError(e)
	}

	// 更新会话成员数（忽略错误）
	conv, _ := dao.ChatConversation.Get(l.ctx, req.ConversationId)
	if conv != nil && conv.MemberCount > 0 {
		_ = dao.ChatConversation.Update(l.ctx, &model.ChatConversation{ID: conv.ID, MemberCount: conv.MemberCount - 1}, "MemberCount")
	}

	// 广播 member_removed
	go func(conversationID uint32, removed string) {
		defer func() { recover() }()
		payload := struct {
			Op   string `json:"op"`
			Data struct {
				ConversationId uint32 `json:"conversationId"`
				RemovedUuid    string `json:"removedUuid"`
			} `json:"data"`
		}{Op: "member_removed"}
		payload.Data.ConversationId = conversationID
		payload.Data.RemovedUuid = removed
		members, err := dao.ChatConversationMember.WithContext(l.ctx).
			Where(dao.ChatConversationMember.ConversationID.Eq(conversationID)).
			Find()
		if err == nil {
			for _, m := range members {
				l.svcCtx.Ws.SendJSON(m.UserUUID, payload)
			}
		}
		// 通知被移除者（可能已不在列表中）
		l.svcCtx.Ws.SendJSON(removed, payload)
	}(req.ConversationId, req.RemoveUUID)

	return nil
}

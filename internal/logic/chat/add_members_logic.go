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

type AddMembersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 群聊添加成员
func NewAddMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddMembersLogic {
	return &AddMembersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddMembersLogic) AddMembers(req *types.AddMembersReq) error {
	// 校验
	if req.UUID == "" || req.ConversationId == 0 || len(req.MemberUUIDs) == 0 {
		return errcode.ErrInvalidParam
	}

	// 校验操作者是会话成员
	if _, e := dao.ChatConversationMember.WithContext(l.ctx).
		Where(
			dao.ChatConversationMember.ConversationID.Eq(req.ConversationId), 
			dao.ChatConversationMember.UserUUID.Eq(req.UUID)).
		Take(); e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return errcode.ErrAuthSession
		}
		return errcode.ErrDataQueryFail.WithError(e)
	}

	// 查询已存在成员集合
	existMembers, e := dao.ChatConversationMember.WithContext(l.ctx).
		Where(dao.ChatConversationMember.ConversationID.Eq(req.ConversationId)).
		Find()
	if e != nil {
		return errcode.ErrDataQueryFail.WithError(e)
	}
	existSet := map[string]struct{}{}
	for _, m := range existMembers {
		existSet[m.UserUUID] = struct{}{}
	}

	// 批量插入新成员（去重、过滤已存在）
	toCreate := make([]*model.ChatConversationMember, 0, len(req.MemberUUIDs))
	addSet := map[string]struct{}{}
	for _, u := range req.MemberUUIDs {
		if u == "" {
			continue
		}
		if _, ok := existSet[u]; ok {
			continue
		}
		if _, ok := addSet[u]; ok {
			continue
		}
		addSet[u] = struct{}{}
		toCreate = append(toCreate, &model.ChatConversationMember{ConversationID: req.ConversationId, UserUUID: u, Role: 1})
	}
	if len(toCreate) > 0 {
		if e := dao.ChatConversationMember.WithContext(l.ctx).CreateInBatches(toCreate, 100); e != nil {
			return errcode.ErrDataCreateFail.WithError(e)
		}
		// 更新成员数（忽略错误）
		_ = dao.ChatConversation.Update(l.ctx, &model.ChatConversation{ID: req.ConversationId, MemberCount: uint32(len(existMembers) + len(toCreate))}, "MemberCount")
	}

	// 广播 member_added 事件给群内所有成员
	go func(conversationID uint32, added []string) {
		defer func() { recover() }()
		payload := struct {
			Op   string `json:"op"`
			Data struct {
				ConversationId uint32   `json:"conversationId"`
				AddedUuids     []string `json:"addedUuids"`
			} `json:"data"`
		}{Op: "member_added"}
		payload.Data.ConversationId = conversationID
		payload.Data.AddedUuids = added
		members, err := dao.ChatConversationMember.WithContext(l.ctx).
			Where(dao.ChatConversationMember.ConversationID.Eq(conversationID)).
			Find()
		if err == nil {
			for _, m := range members {
				l.svcCtx.Ws.SendJSON(m.UserUUID, payload)
			}
		}
		// 给新增成员推送未读为0
		for _, u := range added {
			l.svcCtx.Ws.SendJSON(u, struct {
				Op   string           `json:"op"`
				Data types.UnreadItem `json:"data"`
			}{
				Op: "unread_count_change",
				Data: types.UnreadItem{ConversationId: conversationID, Unread: 0},
			})
		}
	}(req.ConversationId, func() []string { arr := make([]string, 0, len(toCreate)); for _, m := range toCreate { arr = append(arr, m.UserUUID) }; return arr }())

	return nil
}

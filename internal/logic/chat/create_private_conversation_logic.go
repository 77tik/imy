package chat

import (
	"context"
	"errors"
	"sort"
	"strings"

	"imy/internal/dao"
	"imy/internal/dao/model"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type CreatePrivateConversationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建或获取单聊会话
func NewCreatePrivateConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePrivateConversationLogic {
	return &CreatePrivateConversationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreatePrivateConversationLogic) CreatePrivateConversation(req *types.CreatePrivateConversationReq) (resp *types.ConversationInfo, err error) {
	// 参数校验
	if req.UUID == "" || req.PeerUUID == "" || req.UUID == req.PeerUUID {
		return nil, errcode.ErrInvalidParam
	}

	// 规范化单聊 private_key：按字典序拼接，确保唯一且无序
	pair := []string{req.UUID, req.PeerUUID}
	sort.Strings(pair)
	pkey := "p:" + strings.Join(pair, ":")

	// 尝试查询是否已存在
	conv, e := dao.ChatConversation.WithContext(l.ctx).
		Where(dao.ChatConversation.PrivateKey.Eq(pkey)).
		Take()
	if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrDataQueryFail.WithError(e)
	}

	// 创建或补齐成员
	if conv == nil {
		// 创建新会话
		conv = &model.ChatConversation{
			Type:        1,
			PrivateKey:  pkey,
			CreateUUID:  req.UUID,
			Name:        "",
			MemberCount: 2,
			Avatar:      "",
			Extra:       "",
		}
		if ce := dao.ChatConversation.WithContext(l.ctx).Create(conv); ce != nil {
			return nil, errcode.ErrDataCreateFail.WithError(ce)
		}

		// 创建两名成员（创建者与对端）
		members := []*model.ChatConversationMember{
			{ConversationID: conv.ID, UserUUID: req.UUID, Role: 1},
			{ConversationID: conv.ID, UserUUID: req.PeerUUID, Role: 1},
		}
		if me := dao.ChatConversationMember.WithContext(l.ctx).CreateInBatches(members, 2); me != nil {
			return nil, errcode.ErrDataCreateFail.WithError(me)
		}
	} else {
		// 确保两个成员都在会话内（容错补齐）
		for _, u := range []string{req.UUID, req.PeerUUID} {
			if _, me := dao.ChatConversationMember.WithContext(l.ctx).
				Where(
					dao.ChatConversationMember.ConversationID.Eq(conv.ID),
					dao.ChatConversationMember.UserUUID.Eq(u),
				).
				Take(); me != nil {
				if errors.Is(me, gorm.ErrRecordNotFound) {
					_ = dao.ChatConversationMember.WithContext(l.ctx).Create(&model.ChatConversationMember{
						ConversationID: conv.ID,
						UserUUID:       u,
						Role:           1,
					})
					// 尝试更新成员数（忽略错误）
					_ = dao.ChatConversation.Update(l.ctx, &model.ChatConversation{ID: conv.ID, MemberCount: conv.MemberCount + 1}, "MemberCount")
				} else {
					return nil, errcode.ErrDataQueryFail.WithError(me)
				}
			}
		}
	}

	// 组装返回
	resp = &types.ConversationInfo{
		ConversationId: conv.ID,
		Type:           uint32(conv.Type),
		PrivateKey:     conv.PrivateKey,
		Name:           conv.Name,
		MemberCount:    conv.MemberCount,
		LastMessageId:  conv.LastMessageID,
		Avatar:         conv.Avatar,
		Extra:          conv.Extra,
	}

	// 异步推送 WS 通知（新/已有会话都推送，便于客户端刷新）
	go func(info types.ConversationInfo, a, b string) {
		defer func() { recover() }()
		payload := struct {
			Op   string                 `json:"op"`
			Data types.ConversationInfo `json:"data"`
		}{Op: "conversation_created", Data: info}
		l.svcCtx.Ws.SendJSON(a, payload)
		l.svcCtx.Ws.SendJSON(b, payload)
	}( *resp, req.UUID, req.PeerUUID)

	return resp, nil
}

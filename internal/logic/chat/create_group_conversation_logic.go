package chat

import (
	"context"
	"time"

	"imy/internal/dao"
	"imy/internal/dao/model"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"
	"imy/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateGroupConversationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建群聊
func NewCreateGroupConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateGroupConversationLogic {
	return &CreateGroupConversationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateGroupConversationLogic) CreateGroupConversation(req *types.CreateGroupConversationReq) (resp *types.ConversationInfo, err error) {
	// 基础校验
	if req.UUID == "" {
		return nil, errcode.ErrInvalidParam
	}

	// 生成群聊会话
	conv := &model.ChatConversation{
		Type:         2,
		PrivateKey:   "g:" + utils.GenerateUUId(),
		CreateUUID:   req.UUID,
		Name:         req.Name,
		MemberCount:  0, // 稍后根据成员数更新
		Avatar:       "",
		Extra:        "",
		LastMessageID: 0,
	}
	if e := dao.ChatConversation.WithContext(l.ctx).Create(conv); e != nil {
		return nil, errcode.ErrDataCreateFail.WithError(e)
	}

	// 组装成员：创建者 + 传入成员（去重，忽略创建者重复）
	set := map[string]struct{}{req.UUID: {}}
	members := []*model.ChatConversationMember{{ConversationID: conv.ID, UserUUID: req.UUID, Role: 1}}
	for _, u := range req.MemberUUIDs {
		if u == "" {
			continue
		}
		if _, ok := set[u]; ok {
			continue
		}
		set[u] = struct{}{}
		members = append(members, &model.ChatConversationMember{ConversationID: conv.ID, UserUUID: u, Role: 1})
	}
	if len(members) > 0 {
		if e := dao.ChatConversationMember.WithContext(l.ctx).CreateInBatches(members, 100); e != nil {
			return nil, errcode.ErrDataCreateFail.WithError(e)
		}
	}

	// 更新成员数
	conv.MemberCount = uint32(len(members))
	_ = dao.ChatConversation.Update(l.ctx, &model.ChatConversation{ID: conv.ID, MemberCount: conv.MemberCount}, "MemberCount")

	// 装配返回
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

	// 广播创建事件给所有成员
	go func(info types.ConversationInfo, uuids []string) {
		defer func() { recover() }()
		payload := struct {
			Op   string                 `json:"op"`
			Data types.ConversationInfo `json:"data"`
		}{Op: "conversation_created", Data: info}
		for _, id := range uuids {
			l.svcCtx.Ws.SendJSON(id, payload)
		}
		_ = time.Now()
	}( *resp, func() []string { arr := make([]string, 0, len(set)); for k := range set { arr = append(arr, k) }; return arr }())

	return resp, nil
}

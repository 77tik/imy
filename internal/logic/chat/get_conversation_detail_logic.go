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

type GetConversationDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取会话详情
func NewGetConversationDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationDetailLogic {
	return &GetConversationDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetConversationDetailLogic) GetConversationDetail(req *types.GetConversationDetailReq) (resp *types.GetConversationDetailResp, err error) {
	// 参数校验
	if req.UUID == "" || req.ConversationId == 0 {
		return nil, errcode.ErrInvalidParam
	}

	// 校验是否会话成员
	if _, e := dao.ChatConversationMember.WithContext(l.ctx).
		Where(
			dao.ChatConversationMember.ConversationID.Eq(req.ConversationId),
			dao.ChatConversationMember.UserUUID.Eq(req.UUID),
		).
		Take(); e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAuthSession
		}
		return nil, errcode.ErrDataQueryFail.WithError(e)
	}

	// 获取会话信息
	conv, e := dao.ChatConversation.Get(l.ctx, req.ConversationId)
	if e != nil {
		return nil, errcode.ErrDataQueryFail.WithError(e)
	}

	info := types.ConversationInfo{
		ConversationId: conv.ID,
		Type:           uint32(conv.Type),
		PrivateKey:     conv.PrivateKey,
		Name:           conv.Name,
		MemberCount:    conv.MemberCount,
		LastMessageId:  conv.LastMessageID,
		Avatar:         conv.Avatar,
		Extra:          conv.Extra,
	}

	// 获取成员列表
	members, e := dao.ChatConversationMember.WithContext(l.ctx).
		Where(dao.ChatConversationMember.ConversationID.Eq(req.ConversationId)).
		Find()
	if e != nil {
		return nil, errcode.ErrDataQueryFail.WithError(e)
	}

	list := make([]types.ConversationMember, 0, len(members))
	for _, m := range members {
		list = append(list, types.ConversationMember{
			UserUUID:  m.UserUUID,
			Role:      uint32(m.Role),
			Alias:     m.Alias_,
			MuteUntil: m.MuteUntil.UTC().Format(time.RFC3339),
			IsPinned:  func(b bool) uint32 { if b { return 1 }; return 0 }(m.IsPinned),
		})
	}

	return &types.GetConversationDetailResp{Info: info, Members: list}, nil
}

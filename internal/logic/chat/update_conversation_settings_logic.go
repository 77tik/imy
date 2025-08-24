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

type UpdateConversationSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// UpdateSettingsPresenceKey is the context key to carry which fields are present in the incoming JSON.
type UpdateSettingsPresenceKey struct{}

// UpdateSettingsPresence indicates which JSON keys are present in the request body.
type UpdateSettingsPresence struct {
	Alias     bool
	MuteUntil bool
	IsPinned  bool
}

// 更新个人会话设置
func NewUpdateConversationSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateConversationSettingsLogic {
	return &UpdateConversationSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateConversationSettingsLogic) UpdateConversationSettings(req *types.UpdateConversationSettingsReq) error {
	// 参数校验
	if req.UUID == "" || req.ConversationId == 0 {
		return errcode.ErrInvalidParam
	}

	// 校验并获取成员记录
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

	// 提取字段存在性
	presence := UpdateSettingsPresence{}
	if v := l.ctx.Value(UpdateConversationSettingsLogicKey{}); v != nil {
		// backward compatibility if previous key was used
		if p, ok := v.(UpdateSettingsPresence); ok {
			presence = p
		}
	}
	if v := l.ctx.Value(UpdateSettingsPresenceKey{}); v != nil {
		if p, ok := v.(UpdateSettingsPresence); ok {
			presence = p
		}
	}

	// 待更新字段
	cols := make([]string, 0, 3)

	// 别名
	if presence.Alias {
		mem.Alias_ = req.Alias
		cols = append(cols, "Alias_")
	}

	// 免打扰截止时间
	if presence.MuteUntil {
		if req.MuteUntil == "" {
			// 清空为 epoch
			mem.MuteUntil = time.Unix(0, 0).UTC()
		} else {
			t, err := time.Parse(time.RFC3339, req.MuteUntil)
			if err != nil {
				return errcode.ErrInvalidParam
			}
			mem.MuteUntil = t
		}
		cols = append(cols, "MuteUntil")
	}

	// 置顶
	if presence.IsPinned {
		mem.IsPinned = req.IsPinned != 0
		cols = append(cols, "IsPinned")
	}

	if len(cols) == 0 {
		// 无需更新
		return nil
	}

	if e := dao.ChatConversationMember.Update(l.ctx, mem, cols...); e != nil {
		return errcode.ErrDataModifyFail.WithError(e)
	}
	return nil
}

// UpdateConversationSettingsLogicKey is kept for potential backward compatibility if needed.
type UpdateConversationSettingsLogicKey struct{}

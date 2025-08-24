package dao

import (
	"context"

	"imy/internal/dao/model"
)

func (c *chatConversationMember) Example(ctx context.Context) (result *model.ChatConversationMember, err error) {
	// example code
	return c.WithContext(ctx).First()
}

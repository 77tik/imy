package dao

import (
	"context"

	"imy/internal/dao/model"
)

func (c *conversationMember) Example(ctx context.Context) (result *model.ConversationMember, err error) {
	// example code
	return c.WithContext(ctx).First()
}

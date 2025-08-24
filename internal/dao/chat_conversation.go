package dao

import (
	"context"

	"imy/internal/dao/model"
)

func (c *chatConversation) Example(ctx context.Context) (result *model.ChatConversation, err error) {
	// example code
	return c.WithContext(ctx).First()
}

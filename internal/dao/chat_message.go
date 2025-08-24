package dao

import (
	"context"

	"imy/internal/dao/model"
)

func (c *chatMessage) Example(ctx context.Context) (result *model.ChatMessage, err error) {
	// example code
	return c.WithContext(ctx).First()
}

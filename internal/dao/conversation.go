package dao

import (
	"context"

	"imy/internal/dao/model"
)

func (c *conversation) Example(ctx context.Context) (result *model.Conversation, err error) {
	// example code
	return c.WithContext(ctx).First()
}

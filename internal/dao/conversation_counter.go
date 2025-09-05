package dao

import (
	"context"

	"imy/internal/dao/model"
)

func (c *conversationCounter) Example(ctx context.Context) (result *model.ConversationCounter, err error) {
	// example code
	return c.WithContext(ctx).First()
}

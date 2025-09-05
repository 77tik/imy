package dao

import (
	"context"

	"imy/internal/dao/model"
)

func (m *message) Example(ctx context.Context) (result *model.Message, err error) {
	// example code
	return m.WithContext(ctx).First()
}

package dao

import (
	"context"

	"imy/internal/dao/model"
)

func (f *friend) Example(ctx context.Context) (result *model.Friend, err error) {
	// example code
	return f.WithContext(ctx).First()
}

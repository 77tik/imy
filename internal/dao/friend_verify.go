package dao

import (
	"context"

	"imy/internal/dao/model"
)

func (f *friendVerify) Example(ctx context.Context) (result *model.FriendVerify, err error) {
	// example code
	return f.WithContext(ctx).First()
}

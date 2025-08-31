package dao

import (
	"context"

	"imy/internal/dao/model"
)

func (f *friendV2) Example(ctx context.Context) (result *model.FriendV2, err error) {
	// example code
	return f.WithContext(ctx).First()
}

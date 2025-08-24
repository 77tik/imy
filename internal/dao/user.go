package dao

import (
	"context"

	"imy/internal/dao/model"
)

func (u *user) Example(ctx context.Context) (result *model.User, err error) {
	// example code
	return u.WithContext(ctx).First()
}

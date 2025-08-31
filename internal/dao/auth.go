package dao

import (
	"context"

	"imy/internal/dao/model"
	"imy/pkg/dbgen"
)

func (a *auth) Example(ctx context.Context) (result *model.Auth, err error) {
	// example code
	return a.WithContext(ctx).First()
}

func (a *auth) GetByAccount(ctx context.Context, account string, withDeleted ...bool) (result *model.Auth, err error) {
	err = a.DB().WithContext(ctx).Table(model.TableNameAuth).
		Scopes(dbgen.WithDeletedList(withDeleted)).
		Where("account = ?", account).
		First(&result).
		Error
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (a *auth) GetByNickname(ctx context.Context, nickname string, withDeleted ...bool) (result *model.Auth, err error) {
	err = a.DB().WithContext(ctx).Table(model.TableNameAuth).
		Scopes(dbgen.WithDeletedList(withDeleted)).
		Where("nick_name = ?", nickname).
		First(&result).
		Error
	if err != nil {
		return nil, err
	}

	return result, nil
}

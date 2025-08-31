package dao

import (
	"context"

	"imy/internal/dao/model"
	"imy/pkg/dbgen"
)

func (v *verify) Example(ctx context.Context) (result *model.Verify, err error) {
	// example code
	return v.WithContext(ctx).First()
}

func (v *verify) GetListBySendId(ctx context.Context, id uint32, withDeleted ...bool) (list []*model.Verify, err error) {
	err = v.DB().WithContext(ctx).Table(model.TableNameVerify).
		Scopes(dbgen.WithDeletedList(withDeleted)).
		Where("send_id = ?", id).
		Find(&list).
		Error
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (v *verify) GetListByRevId(ctx context.Context, id uint32, withDeleted ...bool) (list []*model.Verify, err error) {
	err = v.DB().WithContext(ctx).Table(model.TableNameVerify).
		Scopes(dbgen.WithDeletedList(withDeleted)).
		Where("rev_id = ?", id).
		Find(&list).
		Error
	if err != nil {
		return nil, err
	}

	return list, nil
}

package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"imy/internal/dao"
	"imy/internal/dao/model"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"
	"imy/pkg/utils"

	"gorm.io/gorm"

	"github.com/zeromicro/go-zero/core/logx"
)

type EmailPasswordRegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 邮箱密码注册
func NewEmailPasswordRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EmailPasswordRegisterLogic {
	return &EmailPasswordRegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EmailPasswordRegisterLogic) EmailPasswordRegister(req *types.EmailPasswordRegisterReq) (resp *types.EmailPasswordRegisterResp, err error) {
	// TODO：验证验证码是否和redis中一致

	// 查一下用户在不在（存在则返回已存在，不存在则继续创建）
	_, err = dao.User.WithContext(l.ctx).Where(dao.User.Email.Eq(req.Email)).Take()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 不存在，继续
		} else {
			return nil, errcode.ErrDataQueryFail.WithError(err)
		}
	} else {
		// 找到了 => 已存在
		return nil, errcode.ErrAuthUserExist
	}

	// 生成密码哈希
	pwdHash, err := utils.PwdGenerate(req.Password)
	if err != nil {
		return nil, errcode.ErrDataCreateFail.WithError(err)
	}

	// 创建用户
	u := &model.User{
		UUID:     utils.GenerateUUId(),
		Email:    req.Email,
		Password: pwdHash,
		NickName: fmt.Sprintf("用户%s", req.Email[:strings.Index(req.Email, "@")]),
		// 其他字段走 DB 默认值或留空
	}
	if err := dao.User.WithContext(l.ctx).Create(u); err != nil {
		return nil, errcode.ErrDataCreateFail.WithError(err)
	}

	// 装配响应
	resp = &types.EmailPasswordRegisterResp{UUID: u.UUID}
	return
}

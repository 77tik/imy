package auth

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"imy/internal/svc"
	"imy/internal/types"
)

type GetEmailCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取邮箱验证码
func NewGetEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEmailCodeLogic {
	return &GetEmailCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetEmailCodeLogic) GetEmailCode(req *types.GetEmailCodeReq) (resp *types.GetEmailCodeResp, err error) {
	// 生成验证码6位
	// code := email.GenerateCode()
	// 发邮件

	// 验证码存redis中

	// redis限制发送速率

	// 返回验证码响应

	return
}

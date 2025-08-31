package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"imy/internal/dao"
	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"
	"imy/pkg/httpx"
	"imy/pkg/jwt"
	"imy/pkg/utils"

	"gorm.io/gorm"

	"github.com/zeromicro/go-zero/core/logx"
)

type EmailPasswordLoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 邮箱密码登陆
func NewEmailPasswordLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EmailPasswordLoginLogic {
	return &EmailPasswordLoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EmailPasswordLoginLogic) EmailPasswordLogin(req *types.EmailPasswordLoginReq) (resp *types.EmailPasswordLoginResp, err error) {
	// 校验邮箱和密码
	if req.Email == "" || req.Password == "" {
		return nil, errcode.ErrAuthInvalidParam
	}

	// 查一下用户存不存在，得到uuid
	u, err := dao.User.WithContext(l.ctx).Where(dao.User.Email.Eq(req.Email)).Take()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAuthUserNotFund
		}
		logx.Errorf("查询用户失败: %v", err)
		return nil, errcode.ErrDataQueryFail.WithError(err)
	}

	// 校验密码
	if err = utils.PwdVerify(req.Password, u.Password); err != nil {
		return nil, errcode.ErrAuthInvalidParam.WithError(err)
	}

	// 生成token，用uuid作为payload
	token, err := jwt.GenToken(jwt.JwtPayLoad{
		Nickname: u.NickName,
		UUID:     u.UUID,
	}, l.svcCtx.Config.Auth.AccessSecret, l.svcCtx.Config.Auth.AccessExpire)
	if err != nil {
		logx.Errorf("生成token失败：%v", err)
		return nil, errcode.ErrAuthTokenFailed.WithError(err)
	}

	// 存入redis会话，如果会话存在就更新成新的，不存在就直接插入
	key := fmt.Sprintf("login_%s", u.UUID)
	session := map[string]any{
		"uuid":        u.UUID,
		"email":       u.Email,
		"nickname":    u.NickName,
		"token":       token,
		"last_active": time.Now().Format("2006-01-02 15:04:05"),
	}
	b, err := json.Marshal(session)
	if err != nil {
		logx.Errorf("序列化会话信息失败：%v", err)
		return nil, errcode.ErrJsonMarshal.WithError(err)
	}
	if err := l.svcCtx.Redis.Set(key, string(b), 24*time.Hour).Err(); err != nil {
		logx.Errorf("存储key于redis失败：%v", err)
		return nil, errcode.ErrRedisSet.WithError(err)
	}

	// 在响应头中写入 token，便于客户端后续携带
	if w, ok := httpx.GetResponse(l.ctx); ok {
		w.Header().Set("token", token)
		w.Header().Set("Authorization", "Bearer "+token)
	}

	// 装配响应
	return &types.EmailPasswordLoginResp{UUID: u.UUID}, nil
}

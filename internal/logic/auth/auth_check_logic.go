package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"imy/internal/errcode"
	"imy/internal/svc"
	"imy/internal/types"
	"imy/pkg/jwt"
	"imy/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AuthCheckLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 身份验证
func NewAuthCheckLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AuthCheckLogic {
	return &AuthCheckLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AuthCheckLogic) AuthCheck(req *types.AuthCheckReq) (resp *types.AuthCheckResp, err error) {
	ok := utils.InListByRegex(l.svcCtx.Config.WhiteList, req.ValidPath)
	if ok {
		logx.Infof("白名单请求：%s", req.ValidPath)
		return
	}
	if req.Token == "" {
		return nil, errcode.ErrAuthTokenNil
	}
	claims, err := jwt.ParseToken(req.Token, l.svcCtx.Config.Auth.AccessSecret)
	if err != nil {
		logx.Errorf("解析token失败：%v", err)
		return nil, errcode.ErrAuthTokenFailed.WithError(err)
	}

	// TODO：设备识别

	// 看一下redis中是否存在token会话记录，如果没有就代表token无用，表示用户没有登陆
	key := fmt.Sprintf("login_%s", claims.UUID)
	loginStr, err := l.svcCtx.Redis.Get(key).Result()
	if err != nil {
		logx.Errorf("解析会话信息失败：%v", err)
		// 返回错误为token无效
		return nil, errcode.ErrAuthTokenUseless.WithError(err)
	}
	// 如果存在，就看一下req的token是否和会话token一致，不一致就代表过期了，也不行
	var loginSession map[string]any
	err = json.Unmarshal([]byte(loginStr), &loginSession)
	if err != nil {
		logx.Errorf("登陆信息格式出错：%v", err)
		return nil, errcode.ErrAuthSession.WithError(err)
	}
	// 必须token一致，然后更新一下最后登陆信息（时间和IP），我们在登出的时候要保存进数据库
	storeToken, ok := loginSession["token"]
	if !ok || req.Token != storeToken.(string) {
		logx.Errorf("token过期：%s", req.Token)
		return nil, errcode.ErrAuthTokenExpire.WithError(err)
	} else {
		loginSession["token"] = req.Token
		loginSession["last_active"] = time.Now().Format("2006-01-02 15:04:05")
		updateSession, err := json.Marshal(loginSession)
		if err != nil {
			logx.Errorf("序列化会话信息失败：%v", err)
			return nil, errcode.ErrJsonMarshal.WithError(err)
		}
		err = l.svcCtx.Redis.Set(key, string(updateSession), 24*time.Hour).Err()
		if err != nil {
			logx.Errorf("存储key于redis失败：%v", err)
			return nil, errcode.ErrRedisSet.WithError(err)
		}
	}

	return &types.AuthCheckResp{
		UUID: claims.UUID,
	}, nil
}

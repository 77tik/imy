package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Config JWT配置
type Config struct {
	SecretKey []byte
	Issuer    string
	ExpiresAt time.Duration
}

// GenerateToken 生成通用JWT令牌
func GenerateToken(config Config, payload map[string]interface{}) (string, error) {
	// 当前时间
	now := time.Now()

	// 创建Claims
	claims := jwt.MapClaims{
		"iat": now.Unix(),                       // 签发时间
		"iss": config.Issuer,                    // 签发者
		"exp": now.Add(config.ExpiresAt).Unix(), // 过期时间
	}

	for k, v := range payload {
		claims[k] = v
	}

	// 创建token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名
	return token.SignedString(config.SecretKey)
}

// jwt v2
type JwtPayLoad struct {
	Nickname string `json:"nickName"`
	UUID     string `json:"uuid"`
}

type CustomClaims struct {
	JwtPayLoad
	jwt.RegisteredClaims
}

func GenToken(payload JwtPayLoad, accessSecret string, expires int) (string, error) {
	claims := CustomClaims{
		JwtPayLoad: payload,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(expires))),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(accessSecret))
}

func ParseToken(tokenStr string, accessSecret string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(accessSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if clains, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return clains, nil
	}
	return nil, err
}

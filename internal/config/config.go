package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf

	Auth        Auth
	Swagger     Swagger
	MySql       MySql
	WorkDir     WorkDir
	WhiteList   []string
	Redis       Redis
	FileServers []FileServer
}

type Auth struct {
	AccessSecret string `json:"AccessSecret"`
	AccessExpire int64  `json:"AccessExpire"`
}

type Swagger struct {
	Host string `json:"Host"`
}

type MySql struct {
	DSN string `json:"DSN,env=MYSQL_DSN"`
}

type WorkDir struct {
	Path string `json:"Path"`
}

type Redis struct {
	Addr     string
	Password string
	DB       int
}
type FileServer struct {
	ApiPrefix string
	Dir       string
}

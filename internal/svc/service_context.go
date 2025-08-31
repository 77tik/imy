package svc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/go-redis/redis"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"imy/internal/config"
	"imy/pkg/dbgen"
	ws "imy/pkg/websocket"
)

// WsHub maintains active websocket connections keyed by uuid.
// It supports register/unregister and JSON push to a user's all connections.
type WsHub struct {
	mu     sync.RWMutex
	byUUID map[string]map[*websocket.Conn]struct{}
	locks  map[*websocket.Conn]*sync.Mutex
}

func NewWsHub() *WsHub {
	return &WsHub{byUUID: make(map[string]map[*websocket.Conn]struct{}), locks: make(map[*websocket.Conn]*sync.Mutex)}
}

func (h *WsHub) getLock(c *websocket.Conn) *sync.Mutex {
	l, ok := h.locks[c]
	if !ok {
		l = &sync.Mutex{}
		h.locks[c] = l
	}
	return l
}

func (h *WsHub) Register(uuid string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.byUUID[uuid]
	if !ok {
		set = make(map[*websocket.Conn]struct{})
		h.byUUID[uuid] = set
	}
	set[conn] = struct{}{}
	if _, ok := h.locks[conn]; !ok {
		h.locks[conn] = &sync.Mutex{}
	}
}

func (h *WsHub) Unregister(uuid string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.byUUID[uuid]; ok {
		delete(set, conn)
		if len(set) == 0 {
			delete(h.byUUID, uuid)
		}
	}
	delete(h.locks, conn)
}

// WithConnWrite acquires the per-connection write lock, executes fn, and releases the lock.
func (h *WsHub) WithConnWrite(conn *websocket.Conn, fn func(*websocket.Conn) error) error {
	h.mu.RLock()
	l, ok := h.locks[conn]
	h.mu.RUnlock()
	if !ok {
		// create lazily
		h.mu.Lock()
		l = h.getLock(conn)
		h.mu.Unlock()
	}
	l.Lock()
	defer l.Unlock()
	return fn(conn)
}

// SendJSON pushes one JSON payload to all active connections of the uuid.
// It removes broken connections on write error.
func (h *WsHub) SendJSON(uuid string, v any) {
	h.mu.RLock()
	set := h.byUUID[uuid]
	h.mu.RUnlock()
	if len(set) == 0 {
		return
	}
	deadline := time.Now().Add(10 * time.Second)
	for c := range set {
		_ = h.WithConnWrite(c, func(conn *websocket.Conn) error {
			_ = conn.SetWriteDeadline(deadline)
			if err := conn.WriteJSON(v); err != nil {
				logx.Infof("ws write error, removing conn: %v", err)
				// cleanup broken conn
				h.mu.Lock()
				if s, ok := h.byUUID[uuid]; ok {
					delete(s, conn)
					if len(s) == 0 {
						delete(h.byUUID, uuid)
					}
				}
				delete(h.locks, conn)
				h.mu.Unlock()
				_ = conn.Close()
			}
			return nil
		})
	}
}

type ServiceContext struct {
	Config config.Config
	Redis  *redis.Client
	Mysql  *gorm.DB
	Ws     *WsHub
	Snow   *snowflake.Node
	WsHub  *ws.Hub
}

func NewServiceContext(c config.Config) *ServiceContext {
	redisClient := InitRedis(c.Redis.Addr, c.Redis.Password, c.Redis.DB)
	// mysql
	mysqldb, err := gorm.Open(mysql.Open(c.MySql.DSN),
		&gorm.Config{
			Logger: dbgen.DefaultLog.LogMode(logger.Info),
		})
	if err != nil {
		logx.Errorf("MySql DSN: %s, err: %s", c.MySql.DSN, err)
		panic("database cannot connected!")
	}
	Node, err := snowflake.NewNode(1)
	if err != nil {
		logx.Errorf("snowflake.NewNode err: %s", err)
	}
	wsHub := ws.NewHub()
	go wsHub.Run()
	return &ServiceContext{
		Config: c,
		Redis:  redisClient,
		Mysql:  mysqldb,
		Ws:     NewWsHub(),
		Snow:   Node,
		WsHub:  wsHub,
	}
}

func InitRedis(addr string, password string, db int) (client *redis.Client) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // no password set
		DB:       db,
		PoolSize: 100,
	})
	_, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := rdb.Ping().Result()
	if err != nil {
		logx.Error(err)
		panic(err)
	}
	fmt.Println("redis链接成功")

	return rdb
}

package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"golang.org/x/time/rate"
	"imy/pkg/jwt"
	"imy/pkg/utils"
)

type GatewayConfig struct {
	rest.RestConf
	Upstream   string            `json:"Upstream"`
	Auth       Auth              `json:"Auth"`
	WhiteList  []string          `json:"WhiteList"`
	Inject     map[string]string `json:"Inject"` // claim -> header name, e.g. {"nickname":"X-User-Nickname"}
	CORS       CORSConfig        `json:"CORS"`
	RateLimit  RateLimitConfig   `json:"RateLimit"`
}

type Auth struct {
	AccessSecret string `json:"AccessSecret"`
	AccessExpire int64  `json:"AccessExpire"`
}

type CORSConfig struct {
	Enabled           bool     `json:"Enabled"`
	AllowOrigins      []string `json:"AllowOrigins"`
	AllowMethods      []string `json:"AllowMethods"`
	AllowHeaders      []string `json:"AllowHeaders"`
	ExposeHeaders     []string `json:"ExposeHeaders"`
	AllowCredentials  bool     `json:"AllowCredentials"`
	MaxAge            int      `json:"MaxAge"`
}

type RateLimitConfig struct {
	Enabled bool    `json:"Enabled"`
	RPS     float64 `json:"RPS"`
	Burst   int     `json:"Burst"`
	Key     string  `json:"Key"` // ip | uuid
}

// simple keyed limiter store
type ClientLimiter struct {
	mu      sync.Mutex
	clients map[string]*rate.Limiter
	rps     rate.Limit
	burst   int
}

func NewClientLimiter(rps float64, burst int) *ClientLimiter {
	if rps <= 0 {
		rps = 10
	}
	if burst <= 0 {
		burst = int(rps * 2)
	}
	return &ClientLimiter{
		clients: make(map[string]*rate.Limiter),
		rps:     rate.Limit(rps),
		burst:  burst,
	}
}

func (c *ClientLimiter) get(key string) *rate.Limiter {
	c.mu.Lock()
	defer c.mu.Unlock()
	lim, ok := c.clients[key]
	if !ok {
		lim = rate.NewLimiter(c.rps, c.burst)
		c.clients[key] = lim
	}
	return lim
}

func (c *ClientLimiter) Allow(key string) bool {
	return c.get(key).Allow()
}

var configFile = flag.String("f", "etc/gateway.yaml", "the config file")

func main() {
	flag.Parse()

	var c GatewayConfig
	conf.MustLoad(*configFile, &c)

	upstreamURL, err := url.Parse(c.Upstream)
	if err != nil {
		panic(fmt.Errorf("invalid upstream url: %w", err))
	}

	// init limiter if enabled
	var limiter *ClientLimiter
	if c.RateLimit.Enabled {
		limiter = NewClientLimiter(c.RateLimit.RPS, c.RateLimit.Burst)
	}

	proxy := httputil.NewSingleHostReverseProxy(upstreamURL)
	origDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		// keep path/query, just rewrite scheme/host and optional base path
		origDirector(r)
		r.URL.Scheme = upstreamURL.Scheme
		r.URL.Host = upstreamURL.Host
		if upstreamURL.Path != "" && upstreamURL.Path != "/" {
			r.URL.Path = singleJoiningSlash(upstreamURL.Path, r.URL.Path)
		}
		// present as upstream host
		r.Host = upstreamURL.Host
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// CORS handling (includes preflight)
		if c.CORS.Enabled {
			writeCORSHeaders(w, r, &c.CORS)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		// Rate limiting (pre-auth by IP)
		if limiter != nil {
			ip := getClientIP(r)
			if ip == "" {
				ip = "unknown"
			}
			if !limiter.Allow("ip:" + ip) {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
		}

		path := r.URL.Path

		// whitelist: pass through without auth
		isWhitelisted := utils.InListByRegex(c.WhiteList, path)
		logx.Infof("Path %s whitelist check: %t", path, isWhitelisted)
		if isWhitelisted {
			logx.Infof("Path %s matched whitelist, bypassing auth", path)
			proxy.ServeHTTP(w, r)
			return
		}

		// extract token
		logx.Infof("Path %s requires auth, extracting token", path)
		token := extractToken(r)
		logx.Infof("Extracted token: %s", token[:20]+"...")
		if token == "" {
			logx.Errorf("No token found for path %s", path)
			http.Error(w, "Unauthorized: token required", http.StatusUnauthorized)
			return
		}

		logx.Infof("Parsing token with secret: %s", c.Auth.AccessSecret)
		claims, err := jwt.ParseToken(token, c.Auth.AccessSecret)
		if err != nil || claims == nil {
			logx.Errorf("gateway: parse token failed: %v", err)
			http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
			return
		}
		logx.Infof("Token parsed successfully, UUID: %s", claims.UUID)

		// Optional: rate limiting by UUID after auth if configured
		if limiter != nil && strings.ToLower(c.RateLimit.Key) == "uuid" && claims.UUID != "" {
			if !limiter.Allow("uuid:" + claims.UUID) {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
		}

		// inject required and configured headers
		// Always override client-provided identity headers
		r.Header.Del("uuid")
		
		r.Header.Set("uuid", claims.UUID)
		logx.Infof("Set UUID header: %s", claims.UUID)

		// Optional: mapping-based injections for extensibility
		for claimKey, headerName := range c.Inject {
			if headerName == "" {
				continue
			}
			val := claimValue(claimKey, claims, token)
			if val != "" {
				r.Header.Set(headerName, val)
			}
		}

		// Ensure request id exists for tracing
		if r.Header.Get("X-Request-Id") == "" {
			r.Header.Set("X-Request-Id", uuid.New().String())
		}

		proxy.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	logx.Infof("Starting gateway at %s -> upstream %s", addr, c.Upstream)
	if err := http.ListenAndServe(addr, nil); err != nil {
		logx.Error(err)
	}
}

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimSpace(auth[len("Bearer "):])
		}
		return strings.TrimSpace(auth)
	}
	return r.Header.Get("token")
}

func claimValue(key string, claims *jwt.CustomClaims, token string) string {
	switch strings.ToLower(key) {
	case "uuid":
		return claims.UUID
	case "nickname", "nick", "nick_name":
		return claims.Nickname
	case "authorization", "auth", "token":
		return "Bearer " + token
	default:
		return ""
	}
}

func singleJoiningSlash(a, b string) string {
	aHas := strings.HasSuffix(a, "/")
	bHas := strings.HasPrefix(b, "/")
	switch {
	case aHas && bHas:
		return a + b[1:]
	case !aHas && !bHas:
		return a + "/" + b
	}
	return a + b
}

func writeCORSHeaders(w http.ResponseWriter, r *http.Request, cfg *CORSConfig) {
	origin := r.Header.Get("Origin")
	allowOrigin := ""
	if len(cfg.AllowOrigins) == 0 || (len(cfg.AllowOrigins) == 1 && cfg.AllowOrigins[0] == "*") {
		// wildcard
		if cfg.AllowCredentials && origin != "" {
			// reflect origin when credentials are allowed
			allowOrigin = origin
			w.Header().Add("Vary", "Origin")
		} else {
			allowOrigin = "*"
		}
	} else {
		for _, o := range cfg.AllowOrigins {
			if strings.EqualFold(o, origin) {
				allowOrigin = origin
				w.Header().Add("Vary", "Origin")
				break
			}
		}
	}
	if allowOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
	}

	methods := "GET,POST,PUT,DELETE,PATCH,OPTIONS"
	if len(cfg.AllowMethods) > 0 {
		methods = strings.Join(cfg.AllowMethods, ",")
	}
	w.Header().Set("Access-Control-Allow-Methods", methods)

	headers := r.Header.Get("Access-Control-Request-Headers")
	if len(cfg.AllowHeaders) > 0 {
		headers = strings.Join(cfg.AllowHeaders, ",")
	}
	if headers != "" {
		w.Header().Set("Access-Control-Allow-Headers", headers)
		w.Header().Add("Vary", "Access-Control-Request-Headers")
	}

	if len(cfg.ExposeHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposeHeaders, ","))
	}
	if cfg.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	if cfg.MaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
	}
}

func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return ip
	}
	return r.RemoteAddr
}
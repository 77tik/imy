package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	pathLogin          = "/api/auth/emailPasswordLogin"
	pathCreatePrivate  = "/api/chat/createPrivate"
	pathSendMessage    = "/api/chat/sendMessage"
	pathGetMessages    = "/api/chat/getMessages"
	pathWS             = "/api/chat/ws"
)

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResp struct {
	UUID string `json:"uuid"`
}

type conversationInfo struct {
	ConversationId uint32 `json:"conversationId"`
	Type           uint32 `json:"type"`
	PrivateKey     string `json:"privateKey"`
	Name           string `json:"name"`
	MemberCount    uint32 `json:"memberCount"`
	LastMessageId  uint64 `json:"lastMessageId"`
	Avatar         string `json:"avatar"`
	Extra          string `json:"extra"`
}

type createPrivateReq struct {
	PeerUUID string `json:"peerUuid"`
}

type sendMessageReq struct {
	ConversationId   uint32   `json:"conversationId"`
	ClientMsgId      string   `json:"clientMsgId"`
	MsgType          uint32   `json:"msgType"`
	Content          string   `json:"content"`
	ContentExtra     string   `json:"contentExtra,omitempty"`
	ReplyToMessageId uint64   `json:"replyToMessageId,omitempty"`
	MentionedUuids   []string `json:"mentionedUuids,omitempty"`
}

type sendMessageResp struct {
	ServerMsgId uint64 `json:"serverMsgId"`
	ClientMsgId string `json:"clientMsgId"`
	CreatedAt   string `json:"createdAt"`
}

type getMessagesReq struct {
	ConversationId uint32 `json:"conversationId"`
	BeforeId       uint64 `json:"beforeId,omitempty"`
	AfterId        uint64 `json:"afterId,omitempty"`
	Limit          int    `json:"limit"`
}

func main() {
	var (
		base    = flag.String("base", "http://127.0.0.1:8081", "Gateway base URL (http[s]://host:port)")
		email   = flag.String("email", "", "Email for login")
		passwd  = flag.String("password", "", "Password for login")
		token   = flag.String("token", "", "Existing JWT token (skip login)")
		peer    = flag.String("peer", "", "Peer UUID to create private conversation with")
		conv    = flag.Uint("conv", 0, "Conversation ID to send message to")
		msg     = flag.String("msg", "", "Message to send (text)")
		wsFlag  = flag.Bool("ws", false, "Connect WebSocket and print incoming messages")
		verbose = flag.Bool("v", false, "Verbose logs")
	)
	flag.Parse()

	logger := log.New(os.Stdout, "[client] ", log.LstdFlags|log.Lmicroseconds)

	if *base == "" {
		logger.Fatal("-base is required")
	}

	cl := &http.Client{Timeout: 15 * time.Second}
	jwt := strings.TrimSpace(*token)
	var userUUID string

	if jwt == "" {
		if *email == "" || *passwd == "" {
			logger.Println("No token provided and email/password missing; attempting login skipped.")
		} else {
			u := strings.TrimRight(*base, "/") + pathLogin
			body, _ := json.Marshal(loginReq{Email: *email, Password: *passwd})
			req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
			if err != nil {
				logger.Fatalf("build login request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := cl.Do(req)
			if err != nil {
				logger.Fatalf("login request failed: %v", err)
			}
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK {
				logger.Fatalf("login non-200: %d, body=%s", resp.StatusCode, string(b))
			}
			// token from headers
			jwt = strings.TrimSpace(resp.Header.Get("Authorization"))
			if strings.HasPrefix(jwt, "Bearer ") {
				jwt = strings.TrimSpace(jwt[len("Bearer "):])
			}
			if jwt == "" {
				jwt = strings.TrimSpace(resp.Header.Get("token"))
			}
			var lr loginResp
			_ = json.Unmarshal(b, &lr)
			userUUID = lr.UUID
			if *verbose {
				logger.Printf("Login OK, uuid=%s, token(prefix)=%s...", userUUID, head(jwt))
			}
		}
	}

	if jwt == "" {
		logger.Println("No token available. Provide -token or -email/-password for login.")
		// still allow proceeding to show usage
	}

	// Optionally create private conversation
	var convID uint32
	if *peer != "" {
		id, err := createPrivateConversation(cl, *base, jwt, *peer, *verbose, logger)
		if err != nil {
			logger.Fatalf("createPrivate: %v", err)
		}
		convID = id
		logger.Printf("Private conversation ready, id=%d", convID)
	}

	// If -conv provided, take it
	if *conv != 0 {
		convID = uint32(*conv)
	}

	// Optionally send a message
	if *msg != "" {
		if convID == 0 {
			logger.Fatal("-msg provided but conversation id is unknown; use -peer or -conv")
		}
		resp, err := sendTextMessage(cl, *base, jwt, convID, *msg, *verbose, logger)
		if err != nil {
			logger.Fatalf("sendMessage: %v", err)
		}
		logger.Printf("Message sent: serverMsgId=%d createdAt=%s", resp.ServerMsgId, resp.CreatedAt)
	}

	if *wsFlag {
		if err := connectWS(*base, jwt, *verbose, logger); err != nil {
			logger.Fatalf("ws: %v", err)
		}
	}
}

func createPrivateConversation(cl *http.Client, base, jwt, peer string, verbose bool, logger *log.Logger) (uint32, error) {
	u := strings.TrimRight(base, "/") + pathCreatePrivate
	payload := createPrivateReq{PeerUUID: peer}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}
	resp, err := cl.Do(req)
	if err != nil {
		return 0, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("non-200: %d, body=%s", resp.StatusCode, string(b))
	}
	var info conversationInfo
	if err := json.Unmarshal(b, &info); err != nil {
		return 0, fmt.Errorf("decode resp: %w", err)
	}
	if verbose {
		logger.Printf("createPrivate resp: %+v", info)
	}
	return info.ConversationId, nil
}

func sendTextMessage(cl *http.Client, base, jwt string, convID uint32, text string, verbose bool, logger *log.Logger) (*sendMessageResp, error) {
	u := strings.TrimRight(base, "/") + pathSendMessage
	payload := sendMessageReq{
		ConversationId: convID,
		ClientMsgId:    randomID(),
		MsgType:        1,
		Content:        text,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}
	resp, err := cl.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200: %d, body=%s", resp.StatusCode, string(b))
	}
	var sr sendMessageResp
	if err := json.Unmarshal(b, &sr); err != nil {
		return nil, fmt.Errorf("decode resp: %w", err)
	}
	if verbose {
		logger.Printf("sendMessage resp: %+v", sr)
	}
	return &sr, nil
}

func connectWS(base, jwt string, verbose bool, logger *log.Logger) error {
	if jwt == "" {
		return fmt.Errorf("jwt token required for WS")
	}
	u, err := url.Parse(strings.TrimRight(base, "/") + pathWS)
	if err != nil {
		return fmt.Errorf("parse base: %w", err)
	}
	// convert http->ws, https->wss
	if u.Scheme == "http" {
		u.Scheme = "ws"
	} else if u.Scheme == "https" {
		u.Scheme = "wss"
	}
	head := http.Header{}
	head.Set("Authorization", "Bearer "+jwt)
	c, _, err := websocket.DefaultDialer.Dial(u.String(), head)
	if err != nil {
		return fmt.Errorf("ws dial: %w", err)
	}
	defer c.Close()
	logger.Printf("WS connected: %s", u.String())

	// read loop
	c.SetReadLimit(1 << 20)
	c.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.SetPongHandler(func(string) error {
		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// simple ping ticker
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				_ = c.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
			}
		}
	}()

	for {
		t, msg, err := c.ReadMessage()
		if err != nil {
			close(done)
			return fmt.Errorf("ws read: %w", err)
		}
		logger.Printf("WS recv type=%d msg=%s", t, string(msg))
		if verbose {
			logger.Printf("raw: %x", msg)
		}
	}
}

func head(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8]
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
)

type model struct {
	base   string
	jwt    string
	uuid   string
	convID uint32
	msgs   []string // activity logs
	input  textinput.Model
	width  int
	height int
	ws     *websocket.Conn
	http   *http.Client
	status string

	// panes data
	convs         []convInfo
	members       []convMember
	chatMsgs      []messageInfo
	selectedIndex int
}

type wsMsg struct{ text string }
type errMsg struct{ err error }

type loginReq struct{ Email, Password string }
type loginResp struct{ UUID string `json:"uuid"` }

// registration & search
type getEmailCodeResp struct{ Code string `json:"code"` }

type registerResp struct{ UUID string `json:"uuid"` }

type searchUserResp struct{ RevId string `json:"revId"` }

type convInfo struct{ ConversationId uint32 `json:"conversationId"`; Type uint32 `json:"type"`; Name string `json:"name"`; MemberCount uint32 `json:"memberCount"`; LastMessageId uint64 `json:"lastMessageId"` }

type convMember struct{ UserUUID string `json:"userUuid"`; Role uint32 `json:"role"`; Alias string `json:"alias"` }

type getConversationsResp struct{ Conversations []convInfo `json:"conversations"` }

type getConversationDetailResp struct{ Info convInfo `json:"info"`; Members []convMember `json:"members"` }

type messageInfo struct{ Id uint64 `json:"id"`; ConversationId uint32 `json:"conversationId"`; SendUuid string `json:"sendUuid"`; MsgType uint32 `json:"msgType"`; Content string `json:"content"`; CreatedAt string `json:"createdAt"` }

type getMessagesResp struct{ Messages []messageInfo `json:"messages"` }

var p *tea.Program

// 持久化配置（仅保存登录态相关）
type persistedCfg struct {
	JWT  string `json:"jwt"`
	UUID string `json:"uuid"`
}

func cfgPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".imy_tui.json"
	}
	return filepath.Join(home, ".imy_tui.json")
}

func loadPersisted() (persistedCfg, error) {
	var c persistedCfg
	b, err := os.ReadFile(cfgPath())
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return persistedCfg{}, err
	}
	return c, nil
}

func savePersisted(c persistedCfg) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath(), b, 0600)
}

func initialModel() model {
	in := textinput.New()
	in.Placeholder = "Type message or /help"
	in.Focus()
	in.CharLimit = 512
	in.Prompt = "> "
	m := model{
		base:  "http://127.0.0.1:8081",
		msgs:  []string{"Welcome to imy TUI. Type /help for commands."},
		input: in,
		http:  &http.Client{Timeout: 12 * time.Second},
	}
	if pc, err := loadPersisted(); err == nil {
		if pc.JWT != "" {
			m.jwt = pc.JWT
		}
		if pc.UUID != "" {
			m.uuid = pc.UUID
		}
	}
	return m
}

func (m model) Init() tea.Cmd { return textinput.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m.quit(), tea.Quit
		case tea.KeyEnter:
			v := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if v == "" {
				return m, nil
			}
			if strings.HasPrefix(v, "/") {
				return m.execCmd(v)
			}
			// send as text message
			if m.convID == 0 {
				m.push("[warn] no conversation. Use /peer <uuid> or /conv <id>.")
				return m, nil
			}
			if m.jwt == "" {
				m.push("[warn] no token. /login or /token <jwt> first.")
				return m, nil
			}
			if err := m.sendMessage(v); err != nil {
				m.push("[send] " + err.Error())
			} else {
				m.push("[me] " + v)
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case wsMsg:
		m.push("[ws] " + msg.text)
		return m, nil
	case errMsg:
		m.push("[err] " + msg.err.Error())
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) View() string {
	// header
	header := fmt.Sprintf("base=%s  uuid=%s  conv=%d  auth=%v", m.base, short(m.uuid), m.convID, m.jwt != "")
	// layout sizes
	contentH := max(1, m.height-3)
	leftW := 28
	rightW := 26
	if m.width > 120 {
		leftW = m.width/4 - 2
		rightW = m.width/4 - 4
	}
	centerW := max(20, m.width-leftW-rightW-2)

	// left: conversations
	var leftLines []string
	leftLines = append(leftLines, lipgloss.NewStyle().Bold(true).Render("Conversations"))
	for i, c := range m.convs {
		name := c.Name
		if name == "" {
			name = fmt.Sprintf("conv-%d", c.ConversationId)
		}
		row := fmt.Sprintf("%d %s (%d)", c.ConversationId, name, c.MemberCount)
		if uint32(i) == m.indexOfConv(m.convID) {
			row = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(row)
		}
		leftLines = append(leftLines, row)
	}
	leftPane := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(leftW).Height(contentH).Padding(0, 1).Render(strings.Join(trimToHeight(leftLines, contentH-2), "\n"))

	// center: chat messages
	var centerLines []string
	centerLines = append(centerLines, lipgloss.NewStyle().Bold(true).Render("Messages"))
	for _, mm := range m.chatMsgs {
		centerLines = append(centerLines, fmt.Sprintf("[%s] %s: %s", mm.CreatedAt, short(mm.SendUuid), mm.Content))
	}
	centerPane := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(centerW).Height(contentH).Padding(0, 1).Render(strings.Join(trimToHeight(centerLines, contentH-2), "\n"))

	// right: members + logs (tail)
	var rightLines []string
	rightLines = append(rightLines, lipgloss.NewStyle().Bold(true).Render("Members"))
	for _, mbr := range m.members {
		rightLines = append(rightLines, fmt.Sprintf("%s r=%d", short(mbr.UserUUID), mbr.Role))
	}
	rightLines = append(rightLines, "--- Logs ---")
	rightLines = append(rightLines, trimToHeight(m.msgs, max(1, contentH-8))...)
	rightPane := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(rightW).Height(contentH).Padding(0, 1).Render(strings.Join(trimToHeight(rightLines, contentH-2), "\n"))

	row := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, centerPane, rightPane)
	return header + "\n" + row + "\n" + m.input.View()
}

func (m *model) push(s string) { m.msgs = append(m.msgs, time.Now().Format("15:04:05 ")+" "+s) }
func (m model) quit() model {
	if m.ws != nil { _ = m.ws.Close() }
	return m
}

func (m model) execCmd(v string) (tea.Model, tea.Cmd) {
	fs := strings.Fields(v)
	if len(fs) == 0 { return m, nil }
	cmd := fs[0]
	switch cmd {
	case "/help":
		m.push("commands: /base <url>, /login <email> <pass>, /code <email>, /register <email> <pass> <code>, /token <jwt>, /search <email>, /peer <uuid>, /group <name> <u1> [u2...], /add <u1> [u2...], /remove <u>, /convs, /detail, /msgs [n], /conv <id>, /refresh, /ws, /clear, /quit")
	case "/base":
		if len(fs) < 2 { m.push("usage: /base http://127.0.0.1:8081"); return m, nil }
		m.base = fs[1]
		m.push("base set")
	case "/login":
		if len(fs) < 3 { m.push("usage: /login <email> <password>"); return m, nil }
		uuid, token, dbg, err := m.login(fs[1], fs[2])
		if err != nil { m.push("login: "+err.Error()); return m, nil }
		m.uuid = uuid
		if strings.TrimSpace(token) == "" {
			_ = savePersisted(persistedCfg{JWT: m.jwt, UUID: m.uuid})
			m.push("login ok. uuid="+short(uuid)+" [warn] no token from headers, use /token <jwt> to set it")
			if dbg != "" { m.push("[debug] "+dbg) }
			return m, nil
		}
		m.jwt = token
		_ = savePersisted(persistedCfg{JWT: m.jwt, UUID: m.uuid})
		m.push("login ok. uuid="+short(uuid)+" token bound")
		m.refreshConvs()
	case "/code":
		if len(fs) != 2 { m.push("usage: /code <email>"); return m, nil }
		code, err := m.getEmailCode(fs[1])
		if err != nil { m.push("getEmailCode: "+err.Error()) } else { m.push("code: "+code) }
	case "/register":
		if len(fs) != 4 { m.push("usage: /register <email> <password> <code>"); return m, nil }
		uuid, err := m.register(fs[1], fs[2], fs[3])
		if err != nil { m.push("register: "+err.Error()) } else { m.push("register ok uuid="+short(uuid)) }
	case "/search":
		if len(fs) != 2 { m.push("usage: /search <email>"); return m, nil }
		rev, err := m.searchUser(fs[1])
		if err != nil { m.push("search: "+err.Error()) } else { m.push("user uuid: "+rev) }
	case "/token":
		if len(fs) < 2 { m.push("usage: /token <jwt>"); return m, nil }
		m.jwt = fs[1]
		_ = savePersisted(persistedCfg{JWT: m.jwt, UUID: m.uuid})
		m.push("token set")
	case "/peer":
		if len(fs) < 2 { m.push("usage: /peer <uuid>"); return m, nil }
		id, err := m.createPrivate(fs[1])
		if err != nil { m.push("createPrivate: "+err.Error()); return m, nil }
		m.setCurrentConv(id)
		m.push(fmt.Sprintf("private ready, id=%d", id))
	case "/group":
		if len(fs) < 3 { m.push("usage: /group <name> <uuid1> [uuid2...]"); return m, nil }
		id, err := m.createGroup(fs[1], fs[2:])
		if err != nil { m.push("createGroup: "+err.Error()); return m, nil }
		m.setCurrentConv(id)
		m.push(fmt.Sprintf("group created, id=%d", id))
	case "/add":
		if m.convID == 0 { m.push("set /conv or create one first"); return m, nil }
		if len(fs) < 2 { m.push("usage: /add <uuid1> [uuid2...]"); return m, nil }
		if err := m.addMembers(m.convID, fs[1:]); err != nil { m.push("addMembers: "+err.Error()) } else { m.push("members added"); m.fetchConvDetailAndMsgs() }
	case "/remove":
		if m.convID == 0 { m.push("set /conv or create one first"); return m, nil }
		if len(fs) != 2 { m.push("usage: /remove <uuid>"); return m, nil }
		if err := m.removeMember(m.convID, fs[1]); err != nil { m.push("removeMember: "+err.Error()) } else { m.push("member removed"); m.fetchConvDetailAndMsgs() }
	case "/convs":
		m.refreshConvs()
	case "/detail":
		if m.convID == 0 { m.push("set /conv or create one first"); return m, nil }
		if err := m.fetchConvDetailAndMsgs(); err != nil { m.push("getConversationDetail: "+err.Error()) }
	case "/msgs":
		if m.convID == 0 { m.push("set /conv or create one first"); return m, nil }
		limit := 20
		if len(fs) == 2 { if n, e := strconv.Atoi(fs[1]); e == nil && n > 0 { limit = n } }
		msgs, err := m.getMessages(m.convID, limit)
		if err != nil { m.push("getMessages: "+err.Error()); return m, nil }
		m.chatMsgs = msgs
	case "/conv":
		if len(fs) < 2 { m.push("usage: /conv <id>"); return m, nil }
		i, err := strconv.Atoi(fs[1])
		if err != nil || i <= 0 { m.push("invalid conv id"); return m, nil }
		m.setCurrentConv(uint32(i))
		m.push("conv set")
	case "/refresh":
		m.refreshConvs()
		if m.convID != 0 { _ = m.fetchConvDetailAndMsgs() }
	case "/ws":
		if m.jwt == "" { m.push("set /token first"); return m, nil }
		if m.ws != nil { _ = m.ws.Close(); m.ws = nil }
		if err := m.connectWS(); err != nil { m.push("ws: "+err.Error()) } else { m.push("ws connected") }
	case "/clear":
		m.msgs = nil
	case "/quit":
		return m.quit(), tea.Quit
	default:
		// send as message if possible
		if m.convID == 0 { m.push("unknown cmd: "+cmd); return m, nil }
		if m.jwt == "" { m.push("[warn] no token. /login or /token <jwt> first."); return m, nil }
		if err := m.sendMessage(strings.TrimSpace(v)); err != nil { m.push("[send] "+err.Error()) } else { m.push("[me] "+strings.TrimSpace(v)); _ = m.fetchConvDetailAndMsgs() }
	}
	return m, nil
}

// -------- HTTP helpers --------
func (m model) doPOST(path string, payload any) ([]byte, http.Header, int, error) {
	u := strings.TrimRight(m.base, "/") + path
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(http.MethodPost, u, body)
	if err != nil { return nil, nil, 0, err }
	if payload != nil { req.Header.Set("Content-Type", "application/json") }
	if m.jwt != "" { req.Header.Set("Authorization", "Bearer "+m.jwt) }
	resp, err := m.http.Do(req)
	if err != nil { return nil, nil, 0, err }
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return b, resp.Header, resp.StatusCode, fmt.Errorf("non-200: %d %s", resp.StatusCode, string(b))
	}
	return b, resp.Header, resp.StatusCode, nil
}

// 禁止自动重定向版本，用于登录以捕获 3xx 响应头（如携带 token）
func (m model) doPOSTNoRedirect(path string, payload any) ([]byte, http.Header, int, error) {
	u := strings.TrimRight(m.base, "/") + path
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(http.MethodPost, u, body)
	if err != nil { return nil, nil, 0, err }
	if payload != nil { req.Header.Set("Content-Type", "application/json") }
	if m.jwt != "" { req.Header.Set("Authorization", "Bearer "+m.jwt) }
	client := *m.http
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil { return nil, nil, 0, err }
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	// 注意：这里不将非 200 视为错误，以便上层可读取 3xx/4xx 的响应头进行诊断
	return b, resp.Header, resp.StatusCode, nil
}

func (m model) login(email, pass string) (uuid, token, debug string, err error) {
	b, h, sc, err := m.doPOSTNoRedirect("/api/auth/emailPasswordLogin", loginReq{Email: email, Password: pass})
	if err != nil { return "", "", "", err }
	// 支持统一包装 {code,msg,data}
	var br struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if sc == http.StatusOK {
		if e := json.Unmarshal(b, &br); e == nil && len(br.Data) > 0 {
			var lr loginResp
			_ = json.Unmarshal(br.Data, &lr)
			uuid = lr.UUID
		} else {
			var lr loginResp
			_ = json.Unmarshal(b, &lr)
			uuid = lr.UUID
		}
	}
	rawAuth := h.Get("Authorization")
	rawTokHdr := h.Get("token")
	tok := strings.TrimSpace(rawAuth)
	if strings.HasPrefix(tok, "Bearer ") { tok = strings.TrimSpace(tok[len("Bearer "):]) }
	if tok == "" { tok = strings.TrimSpace(rawTokHdr) }
	// 仅在无 token 时打印调试信息（不打印敏感值）
	if tok == "" {
		keys := make([]string, 0, len(h))
		for k := range h { keys = append(keys, k) }
		debug = fmt.Sprintf("base=%s status=%d hasAuth=%t hasTokenHeader=%t headerKeys=%v", strings.TrimRight(m.base, "/"), sc, strings.TrimSpace(rawAuth) != "", strings.TrimSpace(rawTokHdr) != "", keys)
	}
	return uuid, tok, debug, nil
}

func (m model) createPrivate(peer string) (uint32, error) {
	payload := map[string]string{"peerUuid": peer}
	b, _, _, err := m.doPOST("/api/chat/createPrivate", payload)
	if err != nil { return 0, err }
	var ci convInfo
	if e := json.Unmarshal(b, &ci); e != nil { return 0, e }
	return ci.ConversationId, nil
}

func (m model) createGroup(name string, members []string) (uint32, error) {
	payload := map[string]any{"name": name, "memberUuids": members}
	b, _, _, err := m.doPOST("/api/chat/createGroup", payload)
	if err != nil { return 0, err }
	var ci convInfo
	if e := json.Unmarshal(b, &ci); e != nil { return 0, e }
	return ci.ConversationId, nil
}

func (m model) addMembers(conv uint32, members []string) error {
	payload := map[string]any{"conversationId": conv, "memberUuids": members}
	_, _, _, err := m.doPOST("/api/chat/addMembers", payload)
	return err
}

func (m model) removeMember(conv uint32, u string) error {
	payload := map[string]any{"conversationId": conv, "removeUuid": u}
	_, _, _, err := m.doPOST("/api/chat/removeMember", payload)
	return err
}

func (m model) getConversations() ([]convInfo, error) {
	payload := map[string]any{"pageSize": 20, "pageIndex": 1}
	b, _, _, err := m.doPOST("/api/chat/getConversations", payload)
	if err != nil { return nil, err }
	var resp getConversationsResp
	if e := json.Unmarshal(b, &resp); e != nil { return nil, e }
	return resp.Conversations, nil
}

func (m model) getConversationDetail(conv uint32) (getConversationDetailResp, error) {
	payload := map[string]any{"conversationId": conv}
	b, _, _, err := m.doPOST("/api/chat/getConversationDetail", payload)
	if err != nil { return getConversationDetailResp{}, err }
	var resp getConversationDetailResp
	if e := json.Unmarshal(b, &resp); e != nil { return getConversationDetailResp{}, e }
	return resp, nil
}

func (m model) getMessages(conv uint32, limit int) ([]messageInfo, error) {
	payload := map[string]any{"conversationId": conv, "limit": limit}
	b, _, _, err := m.doPOST("/api/chat/getMessages", payload)
	if err != nil { return nil, err }
	var resp getMessagesResp
	if e := json.Unmarshal(b, &resp); e != nil { return nil, e }
	return resp.Messages, nil
}

func (m model) sendMessage(text string) error {
	payload := map[string]any{
		"conversationId": m.convID,
		"clientMsgId":    fmt.Sprintf("cli-%d", time.Now().UnixNano()),
		"msgType":        1,
		"content":        text,
	}
	_, _, _, err := m.doPOST("/api/chat/sendMessage", payload)
	return err
}

func (m *model) connectWS() error {
	u, err := url.Parse(strings.TrimRight(m.base, "/") + "/api/chat/ws")
	if err != nil { return err }
	if u.Scheme == "http" { u.Scheme = "ws" } else if u.Scheme == "https" { u.Scheme = "wss" }
	head := http.Header{}
	head.Set("Authorization", "Bearer "+m.jwt)
	c, _, err := websocket.DefaultDialer.Dial(u.String(), head)
	if err != nil { return err }
	m.ws = c
	go func() {
		defer c.Close()
		c.SetReadLimit(1 << 20)
		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.SetPongHandler(func(string) error { c.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })
		// ping
		ping := time.NewTicker(20 * time.Second)
		defer ping.Stop()
		for {
			select {
			case <-ping.C:
				_ = c.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
			default:
				t, msg, err := c.ReadMessage()
				if err != nil { p.Send(errMsg{err}); return }
				p.Send(wsMsg{text: fmt.Sprintf("type=%d %s", t, string(msg))})
			}
		}
	}()
	return nil
}

func short(s string) string { if len(s) > 8 { return s[:8] } ; return s }
func max(a, b int) int { if a > b { return a } ; return b }
// fix: guard negative heights to avoid slice panics
func trimToHeight(ss []string, h int) []string {
	if h <= 0 { return []string{} }
	if len(ss) <= h { return ss }
	return ss[len(ss)-h:]
}

func main() {
	m := initialModel()
	p = tea.NewProgram(m)
	if _, err := p.Run(); err != nil { log.Println("tui error:", err); os.Exit(1) }
}

func (m model) getEmailCode(email string) (string, error) {
	b, _, _, err := m.doPOST("/api/auth/getEmailCode", map[string]string{"email": email})
	if err != nil { return "", err }
	var resp getEmailCodeResp
	if e := json.Unmarshal(b, &resp); e != nil { return "", e }
	return resp.Code, nil
}

func (m model) register(email, pass, code string) (string, error) {
	b, _, _, err := m.doPOST("/api/auth/emailPasswordRegister", map[string]string{"email": email, "password": pass, "code": code})
	if err != nil { return "", err }
	var rr registerResp
	if e := json.Unmarshal(b, &rr); e != nil { return "", e }
	return rr.UUID, nil
}

func (m model) searchUser(email string) (string, error) {
	b, _, _, err := m.doPOST("/api/friend/searchUser", map[string]string{"email": email})
	if err != nil { return "", err }
	var sr searchUserResp
	if e := json.Unmarshal(b, &sr); e != nil { return "", e }
	return sr.RevId, nil
}

func (m *model) setCurrentConv(id uint32) {
	m.convID = id
	m.selectedIndex = int(m.indexOfConv(id))
	_ = m.fetchConvDetailAndMsgs()
}

func (m *model) refreshConvs() {
	convs, err := m.getConversations()
	if err != nil { m.push("getConversations: "+err.Error()); return }
	m.convs = convs
	if m.convID == 0 && len(convs) > 0 {
		m.setCurrentConv(convs[0].ConversationId)
	}
}

func (m *model) fetchConvDetailAndMsgs() error {
	if m.convID == 0 { return nil }
	d, err := m.getConversationDetail(m.convID)
	if err != nil { return err }
	m.members = d.Members
	msgs, err := m.getMessages(m.convID, 20)
	if err != nil { return err }
	m.chatMsgs = msgs
	return nil
}

func (m model) indexOfConv(id uint32) uint32 {
	for i, c := range m.convs {
		if c.ConversationId == id { return uint32(i) }
	}
	return 0
}
package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	"gorm.io/gorm"

	"harmony/component/auth/store"
	"harmony/component/config"
	"harmony/infra/consts"
	"harmony/infra/logger"
	"harmony/infra/model/auth"
	"harmony/infra/model/dict"
	"harmony/infra/model/event"
	"harmony/infra/rdb"
	"harmony/infra/response"
)

const (
	clientMP = "mp"
	clientH5 = "h5"

	ticketWaiting   = "waiting"
	ticketScanned   = "scanned"
	ticketConfirmed = "confirmed"
	ticketCanceled  = "canceled"
	ticketExpired   = "expired"

	ticketTTL      = 2 * time.Minute
	ticketRateTTL  = time.Minute
	ticketRateMax  = 10
	renewThreshold = 7 * 24 * time.Hour
	touchInterval  = time.Hour
	devSecret      = "harmony-dev-secret-please-change-32bytes"
)

type ticketEntry struct {
	Status    string
	UserID    uint64
	Device    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type AuthSrv struct {
	rdb    *rdb.RDB
	store  *store.Store
	wechat WechatClient
	cfg    config.AuthConfig
	wechatCfg config.WechatConfig
	secret []byte
	admins map[uint64]bool
	log    *logger.Logger

	ticketMu sync.Mutex
	tickets  map[string]*ticketEntry
	rate     map[string][]time.Time
}

func NewAuthSrv(
	database *rdb.RDB,
	authStore *store.Store,
	wechat WechatClient,
	cfg config.Config,
) (*AuthSrv, error) {
	secret := cfg.Auth.Secret
	if secret == "" {
		if cfg.Server.Mode != consts.ServerModeDebug {
			return nil, fmt.Errorf("HARMONY_AUTH_SECRET 未设置")
		}
		secret = devSecret
	}
	if len(secret) < 32 {
		return nil, fmt.Errorf("HARMONY_AUTH_SECRET 至少需要 32 个字符")
	}
	admins := make(map[uint64]bool, len(cfg.Auth.AdminUserIDs))
	for _, id := range cfg.Auth.AdminUserIDs {
		admins[id] = true
	}
	return &AuthSrv{
		rdb:       database,
		store:     authStore,
		wechat:    wechat,
		cfg:       cfg.Auth,
		wechatCfg: cfg.Wechat,
		secret:    []byte(secret),
		admins:    admins,
		log:       logger.New(logger.WithModule("auth"), logger.WithSubModule("service")),
		tickets:   make(map[string]*ticketEntry),
		rate:      make(map[string][]time.Time),
	}, nil
}

func (s *AuthSrv) IsAdmin(userID uint64) bool {
	return s.admins[userID]
}

func (s *AuthSrv) WxLogin(ctx context.Context, code string) (*auth.LoginResp, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, response.BadRequest("缺少微信登录 code", "missing code")
	}
	openid, err := s.wechat.Code2Session(ctx, code)
	if err != nil {
		s.log.Error(ctx).Err(err).Msg("code2session failed")
		return nil, response.BadGateway("登录失败，下拉重试")
	}
	var user *auth.User
	err = s.rdb.Transaction(func(tx *gorm.DB) error {
		existing, findErr := s.store.GetUserByOpenID(tx, openid)
		if findErr == nil {
			user = existing
			return nil
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		claimed, claimErr := s.store.ClaimUser(tx, 1, openid)
		if claimErr != nil {
			return claimErr
		}
		if claimed {
			user, claimErr = s.store.GetUser(tx, 1)
			return claimErr
		}
		created := &auth.User{OpenID: &openid, Nickname: "微信用户"}
		if createErr := s.store.CreateUser(tx, created); createErr != nil {
			return createErr
		}
		if initErr := s.initUserData(tx, created.ID); initErr != nil {
			return initErr
		}
		user = created
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.issueToken(ctx, user, clientMP, "微信小程序", s.cfg.MPTokenDays)
}

func (s *AuthSrv) initUserData(tx *gorm.DB, userID uint64) error {
	items := make([]dict.Item, 0, len(consts.BuiltinDictItems))
	for _, seed := range consts.BuiltinDictItems {
		items = append(items, dict.Item{
			DictType: seed.DictType,
			Code:     seed.Code,
			Label:    seed.Label,
			Color:    seed.Color,
			Sort:     seed.Sort,
			Builtin:  true,
		})
	}
	placeholder := event.Event{
		Name:          consts.PlaceholderEventName,
		Type:          consts.EventTypeOther,
		Host:          consts.EventHostOther,
		DatePrecision: consts.DatePrecisionDay,
		Placeholder:   true,
	}
	return s.store.InitUserData(tx, userID, items, placeholder)
}

func (s *AuthSrv) CreateWebTicket(ctx context.Context, ip, device string) (*auth.TicketResp, error) {
	if !s.allowTicket(ip) {
		return nil, response.New(http.StatusTooManyRequests, "登录请求太频繁，请稍后再试", "too many login requests")
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	ticket := hex.EncodeToString(raw)
	expires := time.Now().Add(ticketTTL)
	s.ticketMu.Lock()
	s.tickets[ticket] = &ticketEntry{
		Status:    ticketWaiting,
		Device:    summarizeDevice(device),
		CreatedAt: time.Now(),
		ExpiresAt: expires,
	}
	s.ticketMu.Unlock()

	qrKind := "text"
	qrImage := ""
	if png, err := s.wechat.GetUnlimitedQRCode(ctx, ticket, "sub-auth/confirm", s.wechatCfg.EnvVersion); err == nil && len(png) > 0 {
		qrKind = "wxacode"
		qrImage = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	} else {
		if err != nil {
			s.log.Warn(ctx).Err(err).Msg("generate wxacode failed, fallback to ordinary qr")
		}
		png, qrErr := qrcode.Encode("harmony-login:"+ticket, qrcode.Medium, 512)
		if qrErr != nil {
			return nil, qrErr
		}
		qrImage = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	}
	return &auth.TicketResp{
		Ticket:    ticket,
		QRKind:    qrKind,
		QRImage:   qrImage,
		ExpiresAt: expires.UnixMilli(),
	}, nil
}

func (s *AuthSrv) allowTicket(ip string) bool {
	now := time.Now()
	s.ticketMu.Lock()
	defer s.ticketMu.Unlock()
	items := s.rate[ip]
	kept := items[:0]
	for _, at := range items {
		if now.Sub(at) < ticketRateTTL {
			kept = append(kept, at)
		}
	}
	if len(kept) >= ticketRateMax {
		s.rate[ip] = kept
		return false
	}
	s.rate[ip] = append(kept, now)
	return true
}

func (s *AuthSrv) PollWebTicket(ctx context.Context, ticket string) (*auth.TicketStatusResp, error) {
	s.ticketMu.Lock()
	entry := s.tickets[ticket]
	if entry == nil {
		s.ticketMu.Unlock()
		return &auth.TicketStatusResp{Status: ticketExpired}, nil
	}
	if time.Now().After(entry.ExpiresAt) {
		delete(s.tickets, ticket)
		s.ticketMu.Unlock()
		return &auth.TicketStatusResp{Status: ticketExpired}, nil
	}
	switch entry.Status {
	case ticketConfirmed:
		device := entry.Device
		userID := entry.UserID
		delete(s.tickets, ticket)
		s.ticketMu.Unlock()
		user, err := s.store.GetUser(s.rdb.Get(), userID)
		if err != nil {
			return nil, response.Unauthorized("登录用户不存在")
		}
		login, err := s.issueToken(ctx, user, clientH5, device, s.cfg.WebTokenDays)
		if err != nil {
			return nil, err
		}
		return &auth.TicketStatusResp{Status: ticketConfirmed, Token: login.Token}, nil
	case ticketCanceled:
		delete(s.tickets, ticket)
		s.ticketMu.Unlock()
		return &auth.TicketStatusResp{Status: ticketCanceled}, nil
	default:
		status := entry.Status
		s.ticketMu.Unlock()
		return &auth.TicketStatusResp{Status: status}, nil
	}
}

func (s *AuthSrv) ScanWebTicket(ctx context.Context, userID uint64, ticket string) error {
	s.ticketMu.Lock()
	defer s.ticketMu.Unlock()
	entry := s.tickets[ticket]
	if entry == nil || time.Now().After(entry.ExpiresAt) {
		return response.NotFound("二维码已过期，请在电脑上刷新")
	}
	if entry.Status == ticketConfirmed || entry.Status == ticketCanceled {
		return response.StateConflict("这个二维码已经处理过了")
	}
	if entry.UserID != 0 && entry.UserID != userID {
		return response.StateConflict("这个二维码已被其他微信扫描")
	}
	entry.UserID = userID
	entry.Status = ticketScanned
	return nil
}

func (s *AuthSrv) ConfirmWebTicket(ctx context.Context, userID uint64, ticket string) error {
	s.ticketMu.Lock()
	defer s.ticketMu.Unlock()
	entry := s.tickets[ticket]
	if entry == nil || time.Now().After(entry.ExpiresAt) {
		return response.NotFound("二维码已过期，请在电脑上刷新")
	}
	if entry.Status != ticketScanned || entry.UserID != userID {
		return response.StateConflict("请先在小程序里扫描这个二维码")
	}
	entry.Status = ticketConfirmed
	return nil
}

func (s *AuthSrv) CancelWebTicket(ctx context.Context, userID uint64, ticket string) error {
	s.ticketMu.Lock()
	defer s.ticketMu.Unlock()
	entry := s.tickets[ticket]
	if entry == nil {
		return response.NotFound("二维码已过期，请在电脑上刷新")
	}
	if entry.UserID != 0 && entry.UserID != userID {
		return response.StateConflict("这个二维码不属于当前用户")
	}
	entry.Status = ticketCanceled
	return nil
}

func (s *AuthSrv) VerifyToken(ctx context.Context, rawToken string) (uint64, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return 0, response.Unauthorized("请先登录")
	}
	token, err := s.store.GetTokenByHash(s.rdb.Get(), hashToken(rawToken))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, response.Unauthorized("登录已失效，请重新登录")
		}
		return 0, err
	}
	now := time.Now()
	if now.After(token.ExpiresAt) {
		_ = s.store.DeleteTokenByHash(s.rdb.Get(), token.TokenHash)
		return 0, response.Unauthorized("登录已过期，请重新登录")
	}
	expiresAt := token.ExpiresAt
	lastUsed := token.LastUsedAt
	changed := false
	remaining := expiresAt.Sub(now)
	if remaining < renewThreshold {
		days := s.tokenDays(token.Client)
		expiresAt = now.Add(time.Duration(days) * 24 * time.Hour)
		changed = true
	}
	if now.Sub(lastUsed) > touchInterval {
		lastUsed = now
		changed = true
	}
	if changed {
		if err := s.store.UpdateToken(s.rdb.Get(), token.ID, expiresAt, lastUsed); err != nil {
			return 0, err
		}
	}
	return token.UserID, nil
}

func (s *AuthSrv) Me(ctx context.Context, userID uint64, rawToken string) (*auth.MeResp, error) {
	user, err := s.store.GetUser(s.rdb.Get(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.Unauthorized("登录用户不存在")
		}
		return nil, err
	}
	token, err := s.store.GetTokenByHash(s.rdb.Get(), hashToken(rawToken))
	if err != nil {
		return nil, response.Unauthorized("登录已失效")
	}
	return &auth.MeResp{
		User:    userResp(user, s.IsAdmin(userID)),
		IsAdmin: s.IsAdmin(userID),
		Client:  token.Client,
	}, nil
}

func (s *AuthSrv) Sessions(ctx context.Context, userID uint64) ([]auth.SessionResp, error) {
	tokens, err := s.store.ListTokens(s.rdb.Get(), userID)
	if err != nil {
		return nil, err
	}
	out := make([]auth.SessionResp, 0, len(tokens))
	for i := range tokens {
		out = append(out, auth.SessionResp{
			ID:         tokens[i].ID,
			Client:     tokens[i].Client,
			Device:     tokens[i].Device,
			CreatedAt:  tokens[i].CreatedAt.UnixMilli(),
			LastUsedAt: tokens[i].LastUsedAt.UnixMilli(),
			ExpiresAt:  tokens[i].ExpiresAt.UnixMilli(),
		})
	}
	return out, nil
}

func (s *AuthSrv) RevokeSession(ctx context.Context, userID, id uint64) error {
	return s.store.DeleteToken(s.rdb.Get(), userID, id)
}

func (s *AuthSrv) Logout(ctx context.Context, rawToken string) error {
	return s.store.DeleteTokenByHash(s.rdb.Get(), hashToken(rawToken))
}

func (s *AuthSrv) IssueTokenForCLI(ctx context.Context, userID uint64, days int64) (*auth.LoginResp, error) {
	user, err := s.store.GetUser(s.rdb.Get(), userID)
	if err != nil {
		return nil, err
	}
	if days <= 0 {
		days = s.cfg.WebTokenDays
	}
	return s.issueToken(ctx, user, clientH5, "命令行签发", days)
}

func (s *AuthSrv) issueToken(ctx context.Context, user *auth.User, client, device string, days int64) (*auth.LoginResp, error) {
	if days <= 0 {
		days = s.tokenDays(client)
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	tokenText := base64.RawURLEncoding.EncodeToString(raw)
	now := time.Now()
	expires := now.Add(time.Duration(days) * 24 * time.Hour)
	row := &auth.Token{
		UserID:     user.ID,
		TokenHash:  hashToken(tokenText),
		Client:     client,
		Device:     summarizeDevice(device),
		CreatedAt:  now,
		ExpiresAt:  expires,
		LastUsedAt: now,
	}
	if err := s.store.CreateToken(s.rdb.Get(), row); err != nil {
		return nil, err
	}
	s.log.Info(ctx).Uint64("userId", user.ID).Str("client", client).Msg("auth.tokenIssued")
	return &auth.LoginResp{
		Token:     tokenText,
		ExpiresAt: expires.UnixMilli(),
		User:      userResp(user, s.IsAdmin(user.ID)),
	}, nil
}

func (s *AuthSrv) tokenDays(client string) int64 {
	if client == clientMP {
		return s.cfg.MPTokenDays
	}
	return s.cfg.WebTokenDays
}

func (s *AuthSrv) SignImage(taskID, ownerID uint64) string {
	expires := time.Now().Add(time.Hour).UnixMilli()
	sig := s.imageSig(taskID, ownerID, expires)
	return fmt.Sprintf("/api/v1/recognize/tasks/image?id=%d&exp=%d&sig=%s", taskID, expires, sig)
}

func (s *AuthSrv) VerifyImage(taskID, ownerID uint64, expires int64, sig string) bool {
	if expires <= 0 || time.Now().UnixMilli() > expires {
		return false
	}
	expected := s.imageSig(taskID, ownerID, expires)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func (s *AuthSrv) imageSig(taskID, ownerID uint64, expires int64) string {
	payload := strconv.FormatUint(taskID, 10) + "." + strconv.FormatUint(ownerID, 10) + "." + strconv.FormatInt(expires, 10)
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func userResp(user *auth.User, isAdmin bool) auth.UserResp {
	return auth.UserResp{ID: user.ID, Nickname: user.Nickname, IsAdmin: isAdmin}
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func summarizeDevice(device string) string {
	device = strings.TrimSpace(device)
	if device == "" {
		return ""
	}
	runes := []rune(device)
	if len(runes) > 80 {
		return string(runes[:80])
	}
	return device
}

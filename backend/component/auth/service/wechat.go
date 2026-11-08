package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// WechatClient 封装微信接口，测试时替换为假实现。
type WechatClient interface {
	Code2Session(ctx context.Context, code string) (string, error)
	GetUnlimitedQRCode(ctx context.Context, scene, page, envVersion string) ([]byte, error)
}

type wechatClient struct {
	appID  string
	secret string
	client *http.Client

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func NewWechatClient(appID, secret string) WechatClient {
	return &wechatClient{
		appID:  appID,
		secret: secret,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (w *wechatClient) Code2Session(ctx context.Context, code string) (string, error) {
	if w.appID == "" || w.secret == "" {
		return "", fmt.Errorf("微信 AppID 或 AppSecret 未配置")
	}
	query := url.Values{}
	query.Set("appid", w.appID)
	query.Set("secret", w.secret)
	query.Set("js_code", code)
	query.Set("grant_type", "authorization_code")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.weixin.qq.com/sns/jscode2session?"+query.Encode(), nil)
	if err != nil {
		return "", err
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用微信登录失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var body struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("微信登录返回格式异常: %w", err)
	}
	if body.ErrCode != 0 || body.OpenID == "" {
		return "", fmt.Errorf("微信登录失败: %s", body.ErrMsg)
	}
	return body.OpenID, nil
}

func (w *wechatClient) GetUnlimitedQRCode(ctx context.Context, scene, page, envVersion string) ([]byte, error) {
	token, err := w.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"scene":      scene,
		"page":       page,
		"check_path": false,
		"env_version": envVersion,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.weixin.qq.com/wxa/getwxacodeunlimit?access_token="+url.QueryEscape(token),
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("生成小程序码失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		var failure struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		if err := json.Unmarshal(data, &failure); err == nil && failure.ErrCode != 0 {
			return nil, fmt.Errorf("生成小程序码失败: %s", failure.ErrMsg)
		}
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("生成小程序码失败: 空响应")
	}
	return data, nil
}

func (w *wechatClient) accessToken(ctx context.Context) (string, error) {
	w.mu.Lock()
	if w.token != "" && time.Now().Before(w.expiresAt.Add(-5*time.Minute)) {
		token := w.token
		w.mu.Unlock()
		return token, nil
	}
	appID, secret := w.appID, w.secret
	w.mu.Unlock()

	payload, err := json.Marshal(map[string]any{
		"grant_type":    "client_credential",
		"appid":         appID,
		"secret":        secret,
		"force_refresh": false,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.weixin.qq.com/cgi-bin/stable_token", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取微信 access_token 失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("微信 access_token 返回格式异常: %w", err)
	}
	if body.ErrCode != 0 || body.AccessToken == "" {
		return "", fmt.Errorf("获取微信 access_token 失败: %s", body.ErrMsg)
	}
	expires := time.Duration(body.ExpiresIn) * time.Second
	if expires <= 0 {
		expires = time.Hour
	}
	w.mu.Lock()
	w.token = body.AccessToken
	w.expiresAt = time.Now().Add(expires)
	w.mu.Unlock()
	return body.AccessToken, nil
}

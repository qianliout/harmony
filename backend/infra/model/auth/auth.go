// Package auth 定义用户、登录令牌与扫码 ticket 的出参。
package auth

import "time"

type User struct {
	ID        uint64     `gorm:"primaryKey" json:"id"`
	OpenID    *string    `gorm:"column:openid" json:"openid"`
	UnionID   *string    `gorm:"column:unionid" json:"unionid"`
	Nickname  string     `gorm:"column:nickname" json:"nickname"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"-"`
	UpdatedAt time.Time  `gorm:"column:updated_at" json:"-"`
}

func (*User) TableName() string { return "app_user" }

type Token struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	UserID     uint64    `gorm:"column:user_id" json:"userId"`
	TokenHash  string    `gorm:"column:token_hash" json:"-"`
	Client     string    `gorm:"column:client" json:"client"`
	Device     string    `gorm:"column:device" json:"device"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"-"`
	ExpiresAt  time.Time `gorm:"column:expires_at" json:"-"`
	LastUsedAt time.Time `gorm:"column:last_used_at" json:"-"`
}

func (*Token) TableName() string { return "auth_token" }

type UserResp struct {
	ID       uint64 `json:"id"`
	Nickname string `json:"nickname"`
	IsAdmin  bool   `json:"isAdmin"`
}

type LoginResp struct {
	Token     string   `json:"token"`
	ExpiresAt int64    `json:"expiresAt"`
	User      UserResp `json:"user"`
}

type MeResp struct {
	User   UserResp `json:"user"`
	IsAdmin bool    `json:"isAdmin"`
	Client string   `json:"client"`
}

type SessionResp struct {
	ID         uint64 `json:"id"`
	Client     string `json:"client"`
	Device     string `json:"device"`
	CreatedAt  int64  `json:"createdAt"`
	LastUsedAt int64  `json:"lastUsedAt"`
	ExpiresAt  int64  `json:"expiresAt"`
}

type TicketResp struct {
	Ticket    string `json:"ticket"`
	QRKind    string `json:"qrKind"`
	QRImage   string `json:"qrImage"`
	ExpiresAt int64  `json:"expiresAt"`
}

type TicketStatusResp struct {
	Status string `json:"status"`
	Token  string `json:"token,omitempty"`
}

type WXLoginParam struct {
	Code string `json:"code"`
}

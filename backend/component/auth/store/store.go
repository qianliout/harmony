package store

import (
	"time"

	"gorm.io/gorm"

	"harmony/infra/model/auth"
	"harmony/infra/model/dict"
	"harmony/infra/model/event"
)

type Store struct{}

func New() *Store { return &Store{} }

func (s *Store) GetUserByOpenID(db *gorm.DB, openid string) (*auth.User, error) {
	var user auth.User
	if err := db.Where("openid = ?", openid).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUser(db *gorm.DB, id uint64) (*auth.User, error) {
	var user auth.User
	if err := db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// ClaimUser 把第一个微信 openid 绑定到种子用户 1；已被认领时返回 false。
func (s *Store) ClaimUser(db *gorm.DB, id uint64, openid string) (bool, error) {
	result := db.Model(&auth.User{}).
		Where("id = ? AND (openid IS NULL OR openid = '')", id).
		Updates(map[string]any{"openid": openid, "updated_at": time.Now()})
	return result.RowsAffected > 0, result.Error
}

func (s *Store) CreateUser(db *gorm.DB, user *auth.User) error {
	return db.Create(user).Error
}

func (s *Store) CreateToken(db *gorm.DB, token *auth.Token) error {
	return db.Create(token).Error
}

func (s *Store) GetTokenByHash(db *gorm.DB, hash string) (*auth.Token, error) {
	var token auth.Token
	if err := db.Where("token_hash = ?", hash).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (s *Store) UpdateToken(db *gorm.DB, id uint64, expiresAt, lastUsedAt time.Time) error {
	return db.Model(&auth.Token{}).Where("id = ?", id).Updates(map[string]any{
		"expires_at":   expiresAt,
		"last_used_at": lastUsedAt,
	}).Error
}

func (s *Store) DeleteTokenByHash(db *gorm.DB, hash string) error {
	return db.Where("token_hash = ?", hash).Delete(&auth.Token{}).Error
}

func (s *Store) DeleteToken(db *gorm.DB, userID, id uint64) error {
	return db.Where("user_id = ? AND id = ?", userID, id).Delete(&auth.Token{}).Error
}

func (s *Store) ListTokens(db *gorm.DB, userID uint64) ([]auth.Token, error) {
	items := make([]auth.Token, 0)
	err := db.Where("user_id = ?", userID).Order("last_used_at DESC, id DESC").Find(&items).Error
	return items, err
}

func (s *Store) DeleteExpired(db *gorm.DB, before time.Time) error {
	return db.Where("expires_at < ?", before).Delete(&auth.Token{}).Error
}

// InitUserData 为新用户写入内置字典与占位事项，与 schema.sql 的 owner=1 种子保持一致。
func (s *Store) InitUserData(db *gorm.DB, userID uint64, items []dict.Item, placeholder event.Event) error {
	for i := range items {
		item := items[i]
		item.ID = 0
		item.OwnerID = userID
		if err := db.Create(&item).Error; err != nil {
			return err
		}
	}
	placeholder.ID = 0
	placeholder.OwnerID = userID
	return db.Create(&placeholder).Error
}

package entity

import "time"

type Account struct {
	Base
	Email    string `json:"email" db:"email" gorm:"type:varchar(255);index"`
	Password string `json:"password" db:"password" gorm:"type:varchar(255)"`
}

func (a *Account) TableName() string {
	return "accounts"
}

type Session struct {
	BaseOwned
	ExpiresAt     time.Time `json:"expires_at" db:"expires_at"`
	SomethingElse time.Time `json:"something_else" db:"something_else"`
}

func (s *Session) TableName() string {
	return "sessions"
}

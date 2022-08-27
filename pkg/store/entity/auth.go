package entity

import (
	"time"
)

var _ Model = &Account{}

type Account struct {
	AccountPublic
	Password string ` json:"password" db:"password" gorm:"type:varchar(255)"`
}

type AccountPublic struct {
	Base
	Email      string `json:"email" tsense_default_sort:"1" tsense_sort:"1" db:"email" gorm:"type:varchar(255);index;unique"`
	IsVerified bool   `json:"is_verified" db:"is_verified"`
	IsSSO      bool   `json:"is_sso" db:"is_sso"` // is an sso account
	SSOType    string `json:"sso_type" db:"sso_type"`
}

func (a AccountPublic) GetAccountID() string {
	return a.Base.ID
}

func (a Account) TableName() string {
	return "accounts"
}

type Session struct {
	BaseOwned
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
}

func (s Session) TableName() string {
	return "sessions"
}

const (
	HashVerificationAccountTypeVerify    = "Verify Your Email"
	HashVerificationAccountTypeResetPass = "Reset Your Password"
)

type HashVerificationAccount struct {
	ID          string    `json:"id" db:"id" gorm:"type:varchar(255);primary"`
	AccountID   string    `json:"account_id" db:"account_id" gorm:"type:varchar(255)"`
	Email       string    `json:"email" db:"email" gorm:"type:varchar(255)"`
	TTLExpiry   time.Time `json:"ttl_expiry" db:"ttl_expiry"`
	Type        string    `json:"type" db:"type" gorm:"type:varchar(40)"`
	HasBeenUsed bool      `json:"has_been_used" db:"has_been_used"`
}

func (h HashVerificationAccount) TableName() string {
	return "hash_verification_account"
}

func (h HashVerificationAccount) GetID() string {
	return h.ID
}

// GetIDKey : return the id column name ie your "db" tag
func (h HashVerificationAccount) GetIDKey() string {
	return "id"
}

// GetAccountID : return back the account id for the record
func (h HashVerificationAccount) GetAccountID() string {
	return h.AccountID
}

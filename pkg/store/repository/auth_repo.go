package repository

import (
	"github.com/baderkha/library/pkg/store/entity"
)

type IAccount interface {
	ICrud[entity.Account]
	DoesAccountExist(accountID string, oremail string) bool
	DoesAccountExistByEmail(email string) (bool, *entity.Account)
}

type ISession interface {
	ICrud[entity.Session]
}

type IHashVerificationAccount interface {
	ICrud[entity.HashVerificationAccount]
}

// AccountGorm : gorm account
type AccountGorm struct {
	CrudGorm[entity.Account]
}

func (a *AccountGorm) DoesAccountExist(accountID string, oremail string) bool {
	var c int64
	a.DB.Where("email=?", oremail).Or("account_id=?", accountID).Count(&c)
	return c > 0
}

func (a *AccountGorm) DoesAccountExistByEmail(email string) (bool, *entity.Account) {
	var e entity.Account
	a.DB.Where("email=?", email).First(&e)
	return e.ID != "", &e
}

// SessionGorm : session gorm
type SessionGorm = CrudGorm[entity.Session]
type HashAccountVerification = CrudGorm[entity.HashVerificationAccount]

var _ IAccount = &AccountGorm{}
var _ ISession = &SessionGorm{}
var _ IHashVerificationAccount = &HashAccountVerification{}

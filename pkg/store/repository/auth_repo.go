package repository

import (
	"github.com/baderkha/library/pkg/store/entity"
)

type IAccount interface {
	ICrud[entity.Account]
	DoesAccountExist(accountID string, oremail string) bool
}

type ISession interface {
	ICrud[entity.Session]
}

// AccountGorm : gorm account
type AccountGorm struct {
	CrudGorm[entity.Account]
	BcryptCost int
}

func (a *AccountGorm) DoesAccountExist(accountID string, oremail string) bool {
	var c int64
	a.DB.Where("email=?", oremail).Or("account_id=?", accountID).Count(&c)
	return c > 0
}

// SessionGorm : session gorm
type SessionGorm = CrudGorm[entity.Session]

var _ IAccount = &AccountGorm{}
var _ ISession = &SessionGorm{}

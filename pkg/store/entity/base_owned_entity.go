package entity

// BaseOwned : use this for entities that are owned and need to have an account to own them
type BaseOwned struct {
	Base
	AccountID string `json:"account_id" db:"account_id" gorm:"type:VARCHAR(255);index"`
}

func (b BaseOwned) GetAccountID() string {
	return b.AccountID
}

package entity

// BaseOwnedEntity : use this for entities that are owned and need to have an account to own them
type BaseOwnedEntity struct {
	BaseEntity
	AccountID string `json:"account_id" db:"account_id" gorm:"type:VARCHAR(255);index"`
}

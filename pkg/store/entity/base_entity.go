package entity

import (
	"time"

	"github.com/baderkha/typesense/types"
	"github.com/gofrs/uuid"
)

// Base : attach this as a base model using uuid
type Base struct {
	ID        string          `json:"id" db:"id" gorm:"type:VARCHAR(100);primary"`
	CreatedAt types.Timestamp `json:"created_at" db:"created_at"`
	UpdatedAt types.Timestamp `json:"updated_at" db:"updated_at"`
	IsDeleted bool            `json:"is_deleted" db:"is_deleted" gorm:"type:TINYINT(1);index"`
}

func (b *Base) New() {
	id, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	b.ID = id.String()
	b.IsDeleted = false
	b.CreatedAt = types.Timestamp(time.Now())
	b.UpdatedAt = types.Timestamp(time.Now())
}

func (b Base) GetID() string {
	return b.ID
}

func (b Base) GetIDKey() string {
	return b.ID
}

package entity

import (
	"time"

	"github.com/gofrs/uuid"
)

// BaseEntity : attach this as a base model using uuid
type BaseEntity struct {
	ID        string    `json:"id" db:"id" gorm:"type:VARCHAR(100);primary"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	IsDeleted bool      `json:"is_deleted" db:"is_deleted" gorm:"type:TINYINT(1);index"`
}

func (b *BaseEntity) New() {
	id, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	b.ID = id.String()
	b.IsDeleted = false
	b.CreatedAt = time.Now()
	b.UpdatedAt = time.Now()
}

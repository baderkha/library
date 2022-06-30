package repository

import "gorm.io/gorm"

var _ ITransaction = &GormTransaction{}

type GormTransaction struct {
	DB *gorm.DB
}

func (g *GormTransaction) Begin() ITransaction {
	return &GormTransaction{DB: g.DB.Begin()}
}
func (g *GormTransaction) Commit() error {
	return g.DB.Commit().Error
}
func (g *GormTransaction) RollBack() {
	g.DB.Rollback()
}

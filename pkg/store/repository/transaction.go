package repository

// ITransaction : transaction object for multiple dbs
type ITransaction interface {
	Begin() ITransaction
	Commit() error
	RollBack()
}

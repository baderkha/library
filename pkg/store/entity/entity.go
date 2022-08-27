package entity

// Model : a model definition that can be used by a repository
type Model interface {
	// GetID : return back the id value
	GetID() string
	// GetIDKey : return the id column name ie your "db" tag
	GetIDKey() string // primary index
	// GetAccountID : return back the account id for the record
	GetAccountID() string
	// TableName : return back the table name
	TableName() string // table name or e
}

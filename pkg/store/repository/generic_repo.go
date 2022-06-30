package repository

import "github.com/baderkha/library/pkg/rql"

// Paginated : paginated result
type Paginated[t any] struct {
	CurrentPage  int64 `json:"current_page"`
	CurrentSize  int64 `json:"current_size"`
	Records      []*t  `json:"records"`
	TotalRecords int64 `json:"total_records"`
	TotalPages   int64 `json:"total_pages"`
	IsFinalPage  bool  `json:"is_final_page"`
}

// ICrud : crud interface if your repo is read / write
type ICrud[t any] interface {
	IReadOnly[t]
	IWriteOnly[t]
	// WithTransaction : transactional pointer (make sure all your repos use the same persistence layer)
	WithTransaction(tx ITransaction) ICrud[t]
}

// IReadOnly : repo that only does read operations
type IReadOnly[t any] interface {
	// GetById : get 1 record by id if not found should return err
	GetById(id string) (*t, error)
	// GetAll : get all the records (db dump)
	GetAll() ([]*t, error)
	// GetWithFilterExpression : filter + sort a result using the rql package
	GetWithFilterExpression(f *rql.FilterExpression, s *rql.SortExpression) (data []*t, err error)
	// GetWithFilterExpressionPaginated : filter + sort a result query with pagination using the rql package
	GetWithFilterExpressionPaginated(f *rql.FilterExpression, p *rql.PaginationExpression, s *rql.SortExpression) (data *Paginated[t], err error)
}

// IWriteOnly : repo that only does write operations
type IWriteOnly[t any] interface {
	// Create : create one
	Create(mdl *t) error
	// BulkCreate : create many
	BulkCreate(mdl []*t) error
	// Update : update model
	Update(mdl *t) error
	// DeleteById : perma delete model by id
	DeleteById(id string) error
}

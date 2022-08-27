package repository

import (
	"fmt"
	"math"
	"sync"

	"github.com/baderkha/library/pkg/conditional"
	"github.com/baderkha/library/pkg/rql"
	"github.com/baderkha/library/pkg/store/entity"
	"gorm.io/gorm"
)

const (
	GormBatchSize = 3000
)

type CrudGorm[t entity.Model] struct {
	DB     *gorm.DB
	Parser rql.ISQLFilterParser
	Sorter rql.ISQLSortParser
}

func (c *CrudGorm[t]) Model() t {
	var m t
	return m
}

func (c *CrudGorm[t]) DoesIDExist(id string) bool {
	obj, err := c.GetById(id)
	return err == nil && obj != nil
}

// GetById : get 1 record by id if not found should return err
func (c *CrudGorm[t]) GetById(id string) (*t, error) {
	var res t
	err := c.DB.Table(c.Model().TableName()).Where(c.Model().GetIDKey()+"=?", id).First(&res).Error
	return &res, err
}

// GetAll : get all the records (db dump)
func (c *CrudGorm[t]) GetAll() ([]*t, error) {
	var res []*t
	err := c.DB.Table(c.Model().TableName()).Find(&res).Error
	return res, err
}

func (c *CrudGorm[t]) IsForAccountID(id string, accountID string) bool {
	var count int64
	c.DB.Table(c.Model().TableName()).Where(c.Model().GetIDKey()+"=?", id).Where("account_id=?", accountID).Count(&count)
	return count > 0
}

// GetWithFilterExpression : filter + sort a result using the rql package
func (c *CrudGorm[t]) GetWithFilterExpression(f *rql.FilterExpression, s *rql.SortExpression, baseExpression ...*rql.FilterExpression) (data []*t, err error) {
	var (
		out     rql.SQLOutput
		outBase rql.SQLOutput
		outSort rql.SQLSortOutput
		mdl     t
		schema  = rql.GetSchemaFromTaggedEntity(mdl, "db")
		args    []interface{}
	)
	if f != nil {
		outPtr, err := c.Parser.Parse(f, schema)
		if err != nil {
			return nil, err
		}
		out = *outPtr
		if out.Args != nil && len(out.Args) > 0 {
			args = append(args, out.Args...)
		}
	}
	if len(baseExpression) > 0 && baseExpression[0] != nil {
		outPtr2, err := c.Parser.Parse(baseExpression[0], schema)
		if err != nil {
			return nil, err
		}
		outBase = *outPtr2
		if outBase.Args != nil && len(outBase.Args) > 0 {
			args = append(args, outBase.Args...)
		}
	}
	if s != nil {
		outSPtr, err := c.Sorter.Parse(s, schema)
		if err != nil {
			return nil, err
		}
		outSort = *outSPtr
	}
	out.Query = conditional.Ternary(out.Query == "", " 1=1", out.Query)
	outBase.Query = conditional.Ternary(outBase.Query == "", "1=1", outBase.Query)
	sql := fmt.Sprintf("SELECT * FROM %s WHERE 1=1 AND %s AND %s %s", c.Model().TableName(), out.Query, outBase.Query, outSort.RawQuery)
	err = c.DB.Raw(sql, args...).Find(&data).Error
	return data, err
}

// GetWithFilterExpressionPaginated : filter + sort a result query with pagination using the rql package
func (c *CrudGorm[t]) GetWithFilterExpressionPaginated(f *rql.FilterExpression, p *rql.PaginationExpression, s *rql.SortExpression, baseExpression ...*rql.FilterExpression) (data *Paginated[t], err error) {
	var (
		page        int64 = conditional.Ternary(p != nil, int64(p.Page()), 0)
		limit       int64 = conditional.Ternary(p != nil, int64(p.Size()), 10)
		offset      int64 = conditional.Ternary(p != nil, int64(p.Size()*p.Page()), 0)
		limitClause       = fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)

		out     rql.SQLOutput
		outBase rql.SQLOutput
		outSort rql.SQLSortOutput

		mdl    t
		schema = rql.GetSchemaFromTaggedEntity(mdl, "db")

		args    []interface{}
		records []*t

		mu sync.Mutex
		wg sync.WaitGroup

		count int64
		res   Paginated[t]
	)
	if f != nil {
		outPtr, err := c.Parser.Parse(f, schema)
		if err != nil {
			return nil, err
		}
		out = *outPtr
		if out.Args != nil && len(out.Args) > 0 {
			args = append(args, out.Args...)
		}
	}
	if len(baseExpression) > 0 && baseExpression[0] != nil {
		outPtr2, err := c.Parser.Parse(baseExpression[0], schema)
		if err != nil {
			return nil, err
		}
		outBase = *outPtr2
		if outBase.Args != nil && len(outBase.Args) > 0 {
			args = append(args, outBase.Args...)
		}
	}
	if s != nil {
		outSPtr, err := c.Sorter.Parse(s, schema)
		if err != nil {
			return nil, err
		}
		outSort = *outSPtr
	}

	out.Query = conditional.Ternary(out.Query == "", "1=1", out.Query)
	outBase.Query = conditional.Ternary(outBase.Query == "", "1=1", outBase.Query)

	wg.Add(1)
	go func() {
		defer wg.Done()

		sql := fmt.Sprintf("SELECT * FROM %s WHERE 1=1 AND %s AND %s %s %s", c.Model().TableName(), out.Query, outBase.Query, outSort.RawQuery, limitClause)
		err = c.DB.Raw(sql, args...).Find(&records).Error

		mu.Lock()
		res.Records = records
		mu.Unlock()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()

		sql := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE 1=1 AND %s AND %s", c.Model().TableName(), out.Query, outBase.Query)
		_ = c.DB.Raw(sql, args...).Find(&count)

		mu.Lock()
		res.TotalPages = int64(math.Ceil(float64(count / limit)))
		res.CurrentPage = page
		res.CurrentSize = limit
		res.TotalRecords = count
		res.IsFinalPage = res.CurrentPage >= res.TotalPages
		mu.Unlock()
	}()
	wg.Wait()

	return &res, err
}

// Create : create one
func (c *CrudGorm[t]) Create(mdl *t) error {
	return c.DB.Table(c.Model().TableName()).Create(mdl).Error
}

// BulkCreate : create many
func (c *CrudGorm[t]) BulkCreate(mdl []*t) error {
	return c.DB.Table(c.Model().TableName()).CreateInBatches(mdl, GormBatchSize).Error
}

// Update : update model
func (c *CrudGorm[t]) Update(mdl *t) error {
	return c.DB.Table(c.Model().TableName()).Updates(mdl).Error
}

// DeleteById : perma delete model by id
func (c *CrudGorm[t]) DeleteById(id string) error {
	var res t

	err := c.DB.Table(c.Model().TableName()).Where(c.Model().GetIDKey()+"=?", id).Delete(&res).Error
	return err
}

func (c *CrudGorm[t]) DeleteByIds(id []string) error {
	var res t
	err := c.DB.Table(c.Model().TableName()).Where(c.Model().GetIDKey()+" in (?)", id).Delete(&res).Error
	return err
}

func (c *CrudGorm[t]) WithTransaction(tx ITransaction) ICrud[t] {
	dbtx := tx.(*GormTransaction)
	return &CrudGorm[t]{
		DB: dbtx.DB,
	}
}

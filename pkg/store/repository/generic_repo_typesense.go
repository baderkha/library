package repository

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"github.com/baderkha/library/pkg/conditional"
	"github.com/baderkha/library/pkg/ptr"
	"github.com/baderkha/library/pkg/rql"
	"github.com/baderkha/library/pkg/store/entity"
	"github.com/baderkha/typesense"
	"github.com/wlredeye/jsonlines"
)

type CrudTypeSense[t entity.Model] struct {
	client typesense.IClient[t]
	parser rql.ITypeSenseFilterParser
	sorter rql.ITypeSenseSortParser
}

func (c *CrudTypeSense[t]) Model() t {
	var m t
	return m
}

func (c *CrudTypeSense[t]) Document() typesense.IDocumentClient[t] {
	return c.
		Document().
		WithCollectionName(
			c.Model().TableName(),
		)
}

func (c *CrudTypeSense[t]) Search() typesense.ISearchClient[t] {
	return c.
		Search().
		WithCollectionName(
			c.Model().TableName(),
		)
}

// WithTransaction : transactional pointer (make sure all your repos use the same persistence layer)
func (c *CrudTypeSense[t]) WithTransaction(tx ITransaction) ICrud[t] {
	panic("transactons are not supported with typesense for now")
}

func (c *CrudTypeSense[t]) IsForAccountID(id string, accountID string) bool {
	res, err := c.Document().GetById(id)
	if err != nil {
		return false
	}
	return res != nil && (*res).GetAccountID() == accountID
}

func (c *CrudTypeSense[t]) DoesIDExist(id string) bool {
	res, err := c.Document().GetById(id)
	if err != nil {
		return false
	}
	return res != nil
}

// GetById : get 1 record by id if not found should return err
func (c *CrudTypeSense[t]) GetById(id string) (*t, error) {
	res, err := c.Document().GetById(id)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GetAll : get all the records (db dump)
func (c *CrudTypeSense[t]) GetAll() ([]*t, error) {
	all, err := c.Document().ExportAll()
	if err != nil {
		return nil, err
	}

	return c.fromJSONLines(all)
}

func (c *CrudTypeSense[t]) fromJSONLines(all []byte) ([]*t, error) {
	var res []*t
	var buf bytes.Buffer
	_, err := buf.Read(all)
	if err != nil {
		return nil, err
	}
	err = jsonlines.Decode(&buf, &res)
	return res, err
}

// GetWithFilterExpression : filter + sort a result using the rql package
func (c *CrudTypeSense[t]) GetWithFilterExpression(f *rql.FilterExpression, s *rql.SortExpression, baseExpression ...*rql.FilterExpression) (data []*t, err error) {
	var (
		schema = rql.GetSchemaFromTaggedEntity(ptr.EmptyNonPtr[t](), "db")
		b      = ptr.Empty[rql.FilterExpression]()
	)

	f = ptr.Default(f)
	out, err := c.parser.Parse(f, schema)
	if err != nil {
		return nil, err
	}
	out2, err := c.parser.Parse(b, schema)
	if err != nil {
		return nil, err
	}
	if len(baseExpression) > 0 {
		b = ptr.Default(baseExpression[0])
	}
	all, err := c.Document().ExportAllWithQuery(strings.Join([]string{out.FilterBy, out2.FilterBy}, "&&"))
	return c.fromJSONLines(all)
}

// GetWithFilterExpressionPaginated : filter + sort a result query with pagination using the rql package
func (c *CrudTypeSense[t]) GetWithFilterExpressionPaginated(f *rql.FilterExpression, p *rql.PaginationExpression, s *rql.SortExpression, baseExpression ...*rql.FilterExpression) (data *Paginated[t], err error) {
	p = ptr.Default(p)
	f = ptr.Default(f)
	var (
		schema     = rql.GetSchemaFromTaggedEntity(ptr.EmptyNonPtr[t](), "db")
		b          = ptr.Empty[rql.FilterExpression]()
		res        = ptr.Default(data)
		page   int = conditional.Ternary(p != nil, int(p.Page()), 0)
		limit  int = conditional.Ternary(p != nil, int(p.Size()), 10)
	)

	out, err := c.parser.Parse(f, schema)
	if err != nil {
		return nil, err
	}
	out2, err := c.parser.Parse(b, schema)
	if err != nil {
		return nil, err
	}
	if len(baseExpression) > 0 {
		b = ptr.Default(baseExpression[0])
	}
	out = out.AddPage(page).AddPerPage(limit).AddFilterBy(strings.Join([]string{out.FilterBy, out2.FilterBy}, "&&"))

	all, err := c.Search().Search(out)

	if err != nil {
		return nil, err
	}

	res.CurrentPage = int64(page)
	res.CurrentSize = int64(limit)
	res.TotalRecords = int64(all.OutOf)
	res.TotalPages = int64(math.Ceil(float64(all.OutOf / limit)))
	res.Records = all.GetDocuments()

	return res, nil
}

// Create : create one
func (c *CrudTypeSense[t]) Create(mdl *t) error {
	err := c.Document().Index(mdl)
	if err != nil {
		return err
	}
	return nil
}

// BulkCreate : create many
func (c *CrudTypeSense[t]) BulkCreate(mdl []*t) error {
	err := c.Document().IndexMany(mdl, typesense.DocumentActionUpsert)
	if err != nil {
		return err
	}
	return nil
}

// Update : update model
func (c *CrudTypeSense[t]) Update(mdl *t) error {
	err := c.Document().Update(mdl, (*mdl).GetID())
	if err != nil {
		return err
	}
	return nil
}

// DeleteById : perma delete model by id
func (c *CrudTypeSense[t]) DeleteById(id string) error {
	err := c.Document().DeleteById(id)
	if err != nil {
		return err
	}
	return nil
}

// DeleteByIds : perma delet by many ids
func (c *CrudTypeSense[t]) DeleteByIds(id []string) error {
	if len(id) == 0 {
		return nil
	}
	err := c.Document().DeleteManyWithQuery(fmt.Sprintf("%s:[%s]", c.Model().GetIDKey(), strings.Join(id, ",")))
	if err != nil {
		return err
	}
	return nil
}

package rql

import (
	"errors"
	"strconv"

	"github.com/baderkha/library/pkg/conditional"
)

var (
	errorPageNotANumber = errors.New("expected page to be a number got something else instead ...")
	errorSizeNotANumber = errors.New("expected size to be a number got something else instead ...")
)

// PaginationExpression : pagination expression
type PaginationExpression struct {
	page int
	size int
}

func (p *PaginationExpression) Page() int {
	return p.page
}

func (p *PaginationExpression) Size() int {
	return p.size
}

func PaginationExpressionFromUserInput(page string, size string) (*PaginationExpression, error) {
	page = conditional.Ternary(page == "", "1", page)
	size = conditional.Ternary(page == "", "10", size)
	p, err := strconv.Atoi(page)
	if err != nil {
		return nil, errorPageNotANumber
	}
	s, err := strconv.Atoi(size)
	if err != nil {
		return nil, errorSizeNotANumber
	}
	return &PaginationExpression{
		page: p,
		size: s,
	}, nil
}

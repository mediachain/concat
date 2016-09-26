package query

import (
	"errors"
)

// Constructor for row selectors; individual RowSelectors should only
// be used in a single threaded context.
type MakeRowSelector func() RowSelector

// A RowSelector extracts the next value from an sql result set
type RowSelector interface {
	Scan(src RowScanner) (interface{}, error)
}

// common interface between sql.Row and sql.Rows
type RowScanner interface {
	Scan(res ...interface{}) error
}

// CompileQuery compiles a query to sql.
// Returns the compiled sql query and a selector constructor for extracting
// values from an sql.Rows result set (or a single Row)
func CompileQuery(q *Query) (string, MakeRowSelector, error) {
	return "", nil, errors.New("CompileQuery: Implement Me!")
}

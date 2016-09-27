package query

import (
	"errors"
)

// A RowSelector extracts the next value from an sql result set
type RowSelector interface {
	Scan(src RowScanner) (interface{}, error)
}

// Common Scan interface between sql.Row and sql.Rows
type RowScanner interface {
	Scan(res ...interface{}) error
}

// CompileQuery compiles a query to sql.
// Returns the compiled sql query and a selector for extracting
// values from an sql result set
// Note: The row selector should be used in single-threaded context
func CompileQuery(q *Query) (string, RowSelector, error) {
	return "", nil, errors.New("CompileQuery: Implement Me!")
}

package query

import (
	"errors"
	"fmt"
	"strings"
)

// A RowSelector extracts the next value from an sql result set
type RowSelector interface {
	Scan(src RowScanner) (interface{}, error)
}

// Common Scan interface between sql.Row and sql.Rows
type RowScanner interface {
	Scan(res ...interface{}) error
}

type QueryCompileError string

func (e QueryCompileError) Error() string {
	return string(e)
}

// CompileQuery compiles a query to sql.
// Returns the compiled sql query and a selector for extracting
// values from an sql result set
// Note: The row selector should be used in single-threaded context
func CompileQuery(q *Query) (string, RowSelector, error) {
	var sqlq string
	switch {
	case isStatementQuery(q):
		sqlq = "SELECT %s FROM Statement"
	case isEnvelopeQuery(q):
		sqlq = "SELECT %s FROM Envelope"
	default:
		sqlq = "SELECT %s FROM Statement JOIN Envelope ON Statement.id = Envelope.id"
	}

	cols, err := compileQueryColumns(q)
	if err != nil {
		return "", nil, err
	}
	sqlq = fmt.Sprintf(sqlq, cols)

	return sqlq, nil, errors.New("CompileQuery: Implement Me!")
}

func compileQueryColumns(q *Query) (string, error) {
	switch sel := q.selector.(type) {
	case SimpleSelector:
		return selectorColumn(sel, selectorColumnSimple), nil

	case CompoundSelector:
		if len(sel) == 1 {
			return selectorColumn(sel[0], selectorColumnSimple), nil
		}

		cols := make([]string, len(sel))
		for x := 0; x < len(sel); x++ {
			cols[x] = selectorColumn(sel[x], selectorColumnCompound)
		}
		return strings.Join(cols, ", "), nil

	case *FunctionSelector:
		col := selectorColumn(sel.sel, selectorColumnFun)
		return fmt.Sprintf("%s(%s)", sel.op, col), nil

	default:
		return "", QueryCompileError(fmt.Sprintf("Unexpected selector type: %T", sel))
	}
}

func selectorColumn(sel SimpleSelector, rename map[string]string) string {
	col, ok := rename[string(sel)]
	if !ok {
		col = string(sel)
	}
	return col
}

var selectorColumnSimple = map[string]string{
	"*":         "data",
	"body":      "data",
	"namespace": "DISTINCT namespace",
	"publisher": "DISTINCT publisher",
	"source":    "DISTINCT source"}

var selectorColumnCompound = map[string]string{
	"*":    "data",
	"body": "data"}

var selectorColumnFun = map[string]string{
	"*":         "*",
	"body":      "data",
	"namespace": "DISTINCT namespace",
	"publisher": "DISTINCT publisher",
	"source":    "DISTINCT source"}

func isStatementQuery(q *Query) bool {
	// namespace = * and only has statement selector (*, id, body) and id criteria
	// id acts as statement column
	return q.namespace == "*" &&
		isStatementSelector(q.selector) &&
		(q.criteria == nil || isStatementCriteria(q.criteria))
}

func isEnvelopeQuery(q *Query) bool {
	// namespace != * or envelope selector or any criteria
	// id acts as envelope column
	return q.namespace != "*" ||
		q.criteria != nil ||
		isEnvelopeSelector(q.selector)
}

var statementSelectorp = map[string]bool{
	"*":    true,
	"body": true,
	"id":   true}

var envelopeSelectorp = map[string]bool{
	"id":        true,
	"publisher": true,
	"namespace": true,
	"source":    true,
	"timestamp": true}

func selectorp(sel QuerySelector, tbl map[string]bool) bool {
	switch sel := sel.(type) {
	case SimpleSelector:
		return tbl[string(sel)]

	case CompoundSelector:
		for _, ssel := range sel {
			if tbl[string(ssel)] {
				return true
			}
		}
		return false

	case *FunctionSelector:
		return tbl[string(sel.sel)]

	default:
		return false
	}
}

func isStatementSelector(sel QuerySelector) bool {
	return selectorp(sel, statementSelectorp)
}

func isEnvelopeSelector(sel QuerySelector) bool {
	return selectorp(sel, envelopeSelectorp)
}

func isStatementCriteria(c QueryCriteria) bool {
	switch c := c.(type) {
	case *ValueCriteria:
		return c.sel == "id"

	case *TimeCriteria:
		return false

	case *CompoundCriteria:
		return isStatementCriteria(c.left) && isStatementCriteria(c.right)

	case *NegatedCriteria:
		return isStatementCriteria(c.e)

	default:
		return false
	}
}

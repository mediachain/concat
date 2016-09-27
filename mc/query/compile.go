package query

import (
	"database/sql"
	"fmt"
	ggproto "github.com/gogo/protobuf/proto"
	pb "github.com/mediachain/concat/proto"
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

	crit, err := compileQueryCriteria(q)
	if err != nil {
		return "", nil, err
	}
	if crit != "" {
		sqlq = fmt.Sprintf("%s WHERE %s", sqlq, crit)
	}

	if q.limit > 0 {
		sqlq = fmt.Sprintf("%s LIMIT %d", sqlq, q.limit)
	}

	rsel, err := compileQueryRowSelector(q)
	if err != nil {
		return "", nil, err
	}

	return sqlq, rsel, nil
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

func compileQueryCriteria(q *Query) (string, error) {
	nscrit := compileNamespaceCriteria(q.namespace)
	if q.criteria == nil {
		return nscrit, nil
	}

	scrit, err := compileSelectorCriteria(q.criteria)
	if err != nil {
		return "", err
	}

	if nscrit != "" {
		return fmt.Sprintf("%s AND %s", nscrit, scrit), nil
	}
	return scrit, nil
}

func compileNamespaceCriteria(ns string) string {
	switch {
	case ns == "*":
		return ""
	case ns[len(ns)-1] == '*':
		pre := ns[:len(ns)-2]
		return fmt.Sprintf("namespace LIKE '%s%%'", pre)
	default:
		return fmt.Sprintf("namespace = '%s'", ns)
	}
}

func compileSelectorCriteria(c QueryCriteria) (string, error) {
	switch c := c.(type) {
	case *ValueCriteria:
		return fmt.Sprintf("%s %s '%s'", c.sel, c.op, c.val), nil

	case *TimeCriteria:
		return fmt.Sprintf("timestamp %s %d", c.op, c.ts), nil

	case *CompoundCriteria:
		left, err := compileSelectorCriteria(c.left)
		if err != nil {
			return "", err
		}

		right, err := compileSelectorCriteria(c.right)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("(%s %s %s)", left, c.op, right), nil

	case *NegatedCriteria:
		expr, err := compileSelectorCriteria(c.e)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("NOT %s", expr), nil

	default:
		return "", QueryCompileError(fmt.Sprintf("Unexpected criteria type: %T", c))
	}
}

func compileQueryRowSelector(q *Query) (RowSelector, error) {
	switch sel := q.selector.(type) {
	case SimpleSelector:
		makef, ok := makeSimpleRowSelector[string(sel)]
		if !ok {
			return nil, QueryCompileError(fmt.Sprintf("Unexpected selector: %s", sel))
		}

		return makef(), nil

	case CompoundSelector:
		srs := make([]SimpleRowSelector, len(sel))
		ptrs := make([]interface{}, len(sel))
		for x, ssel := range sel {
			makef, ok := makeSimpleRowSelector[string(ssel)]
			if !ok {
				return nil, QueryCompileError(fmt.Sprintf("Unexpected selector: %s", sel))
			}
			srs[x] = makef()
			ptrs[x] = srs[x].ptr()
		}

		return &RowSelectCompound{sel, srs, ptrs}, nil

	case *FunctionSelector:
		makef, ok := makeFunRowSelector[sel.op]
		if !ok {
			return nil, QueryCompileError(fmt.Sprintf("Unexpected function: %s", sel.op))
		}

		return makef(), nil

	default:
		return nil, QueryCompileError(fmt.Sprintf("Unexpected selector type: %T", sel))
	}
}

type MakeSimpleRowSelector func() SimpleRowSelector

type SimpleRowSelector interface {
	RowSelector
	ptr() interface{}
	value() (interface{}, error)
}

type RowSelectCompound struct {
	sel  []SimpleSelector
	srs  []SimpleRowSelector
	ptrs []interface{}
}

func (rs *RowSelectCompound) Scan(src RowScanner) (interface{}, error) {
	err := src.Scan(rs.ptrs...)
	if err != nil {
		return nil, err
	}

	obj := make(map[string]interface{})
	for x, sel := range rs.sel {
		val, err := rs.srs[x].value()
		if err != nil {
			return nil, err
		}
		obj[string(sel)] = val
	}

	return obj, nil
}

type RowSelectStatement struct {
	val sql.RawBytes
}

func (rs *RowSelectStatement) ptr() interface{} {
	return &rs.val
}

func (rs *RowSelectStatement) value() (interface{}, error) {
	stmt := new(pb.Statement)
	err := ggproto.Unmarshal(rs.val, stmt)
	if err != nil {
		return nil, err
	}
	return stmt, nil
}

func (rs *RowSelectStatement) Scan(src RowScanner) (interface{}, error) {
	err := src.Scan(&rs.val)
	if err != nil {
		return nil, err
	}
	return rs.value()
}

type RowSelectBody struct {
	RowSelectStatement
}

func (rs *RowSelectBody) value() (interface{}, error) {
	stmt, err := rs.RowSelectStatement.value()
	if err != nil {
		return nil, err
	}

	return stmt.(*pb.Statement).Body, nil
}

func (rs *RowSelectBody) Scan(src RowScanner) (interface{}, error) {
	err := src.Scan(&rs.val)
	if err != nil {
		return nil, err
	}
	return rs.value()
}

type RowSelectString struct {
	val string
}

func (rs *RowSelectString) ptr() interface{} {
	return &rs.val
}

func (rs *RowSelectString) value() (interface{}, error) {
	return rs.val, nil
}

func (rs *RowSelectString) Scan(src RowScanner) (interface{}, error) {
	err := src.Scan(&rs.val)
	if err != nil {
		return nil, err
	}
	return rs.val, nil
}

type RowSelectInt struct {
	val int
}

func (rs *RowSelectInt) ptr() interface{} {
	return &rs.val
}

func (rs *RowSelectInt) value() (interface{}, error) {
	return rs.val, nil
}

func (rs *RowSelectInt) Scan(src RowScanner) (interface{}, error) {
	err := src.Scan(&rs.val)
	if err != nil {
		return nil, err
	}
	return rs.val, nil
}

type RowSelectInt64 struct {
	val int64
}

func (rs *RowSelectInt64) ptr() interface{} {
	return &rs.val
}

func (rs *RowSelectInt64) value() (interface{}, error) {
	return rs.val, nil
}

func (rs *RowSelectInt64) Scan(src RowScanner) (interface{}, error) {
	err := src.Scan(&rs.val)
	if err != nil {
		return nil, err
	}
	return rs.val, nil
}

func makeRowSelectStatement() SimpleRowSelector {
	return &RowSelectStatement{}
}

func makeRowSelectBody() SimpleRowSelector {
	return &RowSelectBody{}
}

func makeRowSelectString() SimpleRowSelector {
	return &RowSelectString{}
}

func makeRowSelectInt() SimpleRowSelector {
	return &RowSelectInt{}
}

func makeRowSelectInt64() SimpleRowSelector {
	return &RowSelectInt64{}
}

var makeSimpleRowSelector = map[string]MakeSimpleRowSelector{
	"*":         makeRowSelectStatement,
	"body":      makeRowSelectBody,
	"id":        makeRowSelectString,
	"namespace": makeRowSelectString,
	"publisher": makeRowSelectString,
	"source":    makeRowSelectString,
	"timestamp": makeRowSelectInt64}

var makeFunRowSelector = map[string]MakeSimpleRowSelector{
	"COUNT": makeRowSelectInt}

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

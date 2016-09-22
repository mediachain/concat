package query

import (
	"fmt"
	pb "github.com/mediachain/concat/proto"
	"strings"
)

// query evaluation: very primitive eval with only simple statements
// will have to do until we have an index and we can compile to sql.
func EvalQuery(query *Query, stmts []*pb.Statement) ([]interface{}, error) {
	nsfilter := makeNamespaceFilter(query)

	cfilter, err := makeCriteriaFilter(query)
	if err != nil {
		return nil, err
	}

	rs, err := makeResultSet(query)
	if err != nil {
		return nil, err
	}

	rs.begin(len(stmts))
	for _, stmt := range stmts {
		if nsfilter(stmt) && cfilter(stmt) {
			rs.add(stmt)
		}
	}
	rs.end()

	return rs.result(), nil
}

type QueryResultSet interface {
	begin(hint int)
	add(*pb.Statement)
	end()
	result() []interface{}
}

type QueryEvalError string

func (e QueryEvalError) Error() string {
	return string(e)
}

type StatementFilter func(*pb.Statement) bool
type ValueCriteriaFilterSelect func(*pb.Statement) string
type ValueCriteriaFilterCompare func(a, b string) bool
type TimeCriteriaFilter func(*pb.Statement, int64) bool
type CompoundCriteriaFilter func(stmt *pb.Statement, left, right StatementFilter) bool

func idCriteriaFilter(stmt *pb.Statement) string {
	return stmt.Id
}

func publisherCriteriaFilter(stmt *pb.Statement) string {
	return stmt.Publisher
}

func sourceCriteriaFilter(stmt *pb.Statement) string {
	// only support simple statements for now, so src = publisher
	return stmt.Publisher
}

var valueCriteriaFilterSelect = map[string]ValueCriteriaFilterSelect{
	"id":        idCriteriaFilter,
	"publisher": publisherCriteriaFilter,
	"source":    sourceCriteriaFilter}

func valueCriteriaEQ(a, b string) bool {
	return a == b
}

func valueCriteriaNE(a, b string) bool {
	return a != b
}

var valueCriteriaFilterCompare = map[string]ValueCriteriaFilterCompare{
	"=":  valueCriteriaEQ,
	"!=": valueCriteriaNE}

func timestampFilterLTEQ(stmt *pb.Statement, ts int64) bool {
	return stmt.Timestamp <= ts
}

func timestampFilterLT(stmt *pb.Statement, ts int64) bool {
	return stmt.Timestamp < ts
}

func timestampFilterEQ(stmt *pb.Statement, ts int64) bool {
	return stmt.Timestamp == ts
}

func timestampFilterNE(stmt *pb.Statement, ts int64) bool {
	return stmt.Timestamp != ts
}

func timestampFilterGTEQ(stmt *pb.Statement, ts int64) bool {
	return stmt.Timestamp >= ts
}

func timestampFilterGT(stmt *pb.Statement, ts int64) bool {
	return stmt.Timestamp > ts
}

var timeCriteriaFilters = map[string]TimeCriteriaFilter{
	"<=": timestampFilterLTEQ,
	"<":  timestampFilterLT,
	"=":  timestampFilterEQ,
	"!=": timestampFilterNE,
	">=": timestampFilterGTEQ,
	">":  timestampFilterGT}

func compoundCriteriaAND(stmt *pb.Statement, left, right StatementFilter) bool {
	return left(stmt) && right(stmt)
}

func compoundCriteriaOR(stmt *pb.Statement, left, right StatementFilter) bool {
	return left(stmt) || right(stmt)
}

var compoundCriteriaFilters = map[string]CompoundCriteriaFilter{
	"AND": compoundCriteriaAND,
	"OR":  compoundCriteriaOR}

func emptyFilter(*pb.Statement) bool {
	return true
}

func makeNamespaceFilter(query *Query) StatementFilter {
	ns := query.namespace
	switch {
	case ns == "*":
		return emptyFilter

	case ns[len(ns)-1] == '*':
		prefix := ns[:len(ns)-2]
		return func(stmt *pb.Statement) bool {
			return strings.HasPrefix(stmt.Namespace, prefix)
		}

	default:
		return func(stmt *pb.Statement) bool {
			return stmt.Namespace == ns
		}
	}
}

func makeCriteriaFilter(query *Query) (StatementFilter, error) {
	c := query.criteria
	if c == nil {
		return emptyFilter, nil
	}

	return makeCriteriaFilterF(c)
}

func makeCriteriaFilterF(c QueryCriteria) (StatementFilter, error) {
	switch c := c.(type) {
	case *ValueCriteria:
		getf, ok := valueCriteriaFilterSelect[c.sel]
		if !ok {
			return nil, QueryEvalError(fmt.Sprintf("Unexpected criteria selector: %s", c.sel))
		}

		cmpf, ok := valueCriteriaFilterCompare[c.op]
		if !ok {
			return nil, QueryEvalError(fmt.Sprintf("Unexpected criteria operator: %s", c.op))
		}

		return func(stmt *pb.Statement) bool {
			return cmpf(getf(stmt), c.val)
		}, nil

	case *TimeCriteria:
		filter, ok := timeCriteriaFilters[c.op]
		if !ok {
			return nil, QueryEvalError(fmt.Sprintf("Unexpected criteria time op: %s", c.op))
		}

		return func(stmt *pb.Statement) bool {
			return filter(stmt, c.ts)
		}, nil

	case *CompoundCriteria:
		filter, ok := compoundCriteriaFilters[c.op]
		if !ok {
			return nil, QueryEvalError(fmt.Sprintf("Unexpected criteria combinator: %s", c.op))
		}

		left, err := makeCriteriaFilterF(c.left)
		if err != nil {
			return nil, err
		}

		right, err := makeCriteriaFilterF(c.right)
		if err != nil {
			return nil, err
		}

		return func(stmt *pb.Statement) bool {
			return filter(stmt, left, right)
		}, nil

	default:
		return nil, QueryEvalError(fmt.Sprintf("Unexpected criteria type: %T", c))
	}
}

type StatementSelector func(*pb.Statement) interface{}

func simpleSelectorAll(stmt *pb.Statement) interface{} {
	return stmt
}

func simpleSelectorBody(stmt *pb.Statement) interface{} {
	return stmt.Body
}

func simpleSelectorId(stmt *pb.Statement) interface{} {
	return stmt.Id
}

func simpleSelectorPublisher(stmt *pb.Statement) interface{} {
	return stmt.Publisher
}

func simpleSelectorNamespace(stmt *pb.Statement) interface{} {
	return stmt.Namespace
}

func simpleSelectorSource(stmt *pb.Statement) interface{} {
	// simple statements, source = publisher
	return stmt.Publisher
}

func simpleSelectorTimestamp(stmt *pb.Statement) interface{} {
	return stmt.Timestamp
}

var simpleSelectors = map[string]StatementSelector{
	"*":         simpleSelectorAll,
	"body":      simpleSelectorBody,
	"id":        simpleSelectorId,
	"publisher": simpleSelectorPublisher,
	"namespace": simpleSelectorNamespace,
	"source":    simpleSelectorSource,
	"timestamp": simpleSelectorTimestamp}

type FunctionStatementSelector func([]interface{}) []interface{}

func countFunctionSelector(res []interface{}) []interface{} {
	return []interface{}{len(res)}
}

var functionSelectors = map[string]FunctionStatementSelector{
	"COUNT": countFunctionSelector}

// The difference between the types of result set:
//  Simple selectors (SimpleResultSet) have unique (set) semantics.
//  Compound selectors (CompoundResultSet) create objects with fields named by
//   their selector, and have distinct semantics.
//  Function selectors (FunctionResultSet) perform a selection and apply a
//   function on the simple result set.
// The difference is illustrated with these two expressions
//  SELECT namespace FROM *
//  SELECT (namespace) FROM *
//  SELECT COUNT(namespace) FROM *
// The first form, will return a set of unique namespaces from all statements
// as flat strings
// The second form will return a list of objects, with a field named namespace
//  containing the namespace of some statement. There are as many objects in
//  the result set as statements.
// The third form will return a list with one element, which will be the count
//  of distinct namespaces.
func makeResultSet(query *Query) (QueryResultSet, error) {
	sel := query.selector
	switch sel := sel.(type) {
	case SimpleSelector:
		getf, ok := simpleSelectors[string(sel)]
		if !ok {
			return nil, QueryEvalError(fmt.Sprintf("Unexpected selector: %s", sel))
		}

		return makeSimpleResultSet(getf, query.limit), nil

	case CompoundSelector:
		getfs := make([]StatementSelector, len(sel))
		for x, key := range sel {
			getf, ok := simpleSelectors[string(key)]
			if !ok {
				return nil, QueryEvalError(fmt.Sprintf("Unexpected selector: %s", key))
			}
			getfs[x] = getf
		}

		return makeCompoundResultSet(sel, getfs, query.limit), nil

	case *FunctionSelector:
		fun, ok := functionSelectors[sel.op]
		if !ok {
			return nil, QueryEvalError(fmt.Sprintf("Unexpected selector: %s", sel.op))
		}

		getf, ok := simpleSelectors[string(sel.sel)]
		if !ok {
			return nil, QueryEvalError(fmt.Sprintf("Unexpected selector: %s", sel.sel))
		}

		return makeFunctionResultSet(fun, getf, query.limit), nil

	default:
		return nil, QueryEvalError(fmt.Sprintf("Unexpected selector type: %T", sel))
	}
}

func makeSimpleResultSet(getf StatementSelector, limit int) QueryResultSet {
	return &SimpleResultSet{rset: make(map[interface{}]bool), getf: getf, limit: limit}
}

type SimpleResultSet struct {
	rset  map[interface{}]bool
	res   []interface{}
	getf  StatementSelector
	limit int
}

func (rs *SimpleResultSet) begin(hint int) {}

func (rs *SimpleResultSet) add(stmt *pb.Statement) {
	if rs.limit == 0 || len(rs.rset) < rs.limit {
		rs.rset[rs.getf(stmt)] = true
	}
}

func (rs *SimpleResultSet) end() {
	rs.res = make([]interface{}, len(rs.rset))
	x := 0
	for obj, _ := range rs.rset {
		rs.res[x] = obj
		x++
	}
	rs.rset = nil
}

func (rs *SimpleResultSet) result() []interface{} {
	return rs.res
}

func makeCompoundResultSet(sels []SimpleSelector, getfs []StatementSelector, limit int) QueryResultSet {
	compf := makeCompoundStatementSelector(sels, getfs)
	return &CompoundResultSet{getf: compf, limit: limit}
}

func makeCompoundStatementSelector(sels []SimpleSelector, getfs []StatementSelector) StatementSelector {
	return func(stmt *pb.Statement) interface{} {
		val := make(map[string]interface{})
		for x, sel := range sels {
			val[string(sel)] = getfs[x](stmt)
		}
		return val
	}
}

type CompoundResultSet struct {
	rset  []interface{}
	getf  StatementSelector
	limit int
}

func (rs *CompoundResultSet) begin(hint int) {
	rs.rset = make([]interface{}, 0, hint)
}

func (rs *CompoundResultSet) add(stmt *pb.Statement) {
	rs.rset = append(rs.rset, rs.getf(stmt))
}

func (rs *CompoundResultSet) end() {}

func (rs *CompoundResultSet) result() []interface{} {
	return rs.rset
}

func makeFunctionResultSet(fun FunctionStatementSelector, getf StatementSelector, limit int) QueryResultSet {
	return &FunctionResultSet{rset: makeSimpleResultSet(getf, limit), fun: fun}
}

type FunctionResultSet struct {
	rset QueryResultSet
	fun  FunctionStatementSelector
	res  []interface{}
}

func (rs *FunctionResultSet) begin(hint int) {
	rs.rset.begin(hint)
}

func (rs *FunctionResultSet) add(stmt *pb.Statement) {
	rs.rset.add(stmt)
}

func (rs *FunctionResultSet) end() {
	rs.rset.end()
	rs.res = rs.fun(rs.rset.result())
}

func (rs *FunctionResultSet) result() []interface{} {
	return rs.res
}

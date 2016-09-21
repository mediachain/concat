package main

import (
	"fmt"
	pb "github.com/mediachain/concat/proto"
	"strconv"
	"strings"
)

type Query struct {
	namespace string
	selector  QuerySelector
	criteria  QueryCriteria
	limit     int
}

type QuerySelector interface {
	SelectorType() string
}

type SimpleSelector string
type CompoundSelector []SimpleSelector
type FunctionSelector struct {
	op  string
	sel SimpleSelector
}

func (s SimpleSelector) SelectorType() string {
	return "simple"
}

func (s CompoundSelector) SelectorType() string {
	return "compound"
}

func (s *FunctionSelector) SelectorType() string {
	return "function"
}

type QueryCriteria interface {
	CriteriaType() string
}

type ValueCriteria struct {
	sel string
	val string
}

type TimeCriteria struct {
	op string
	ts int64
}

type CompoundCriteria struct {
	op          string
	left, right QueryCriteria
}

func (c *ValueCriteria) CriteriaType() string {
	return "value"
}

func (c *TimeCriteria) CriteriaType() string {
	return "time"
}

func (c *CompoundCriteria) CriteriaType() string {
	return "compound"
}

type ConsCell struct {
	car interface{}
	cdr *ConsCell
}

type ParseState struct {
	query *Query
	stack *ConsCell
	err   error
}

func (ps *ParseState) setSimpleSelector() {
	// stack: simple-selector
	sel := ps.pop().(string)
	ps.query.selector = SimpleSelector(sel)
}

func (ps *ParseState) setCompoundSelector() {
	// stack: simple-selector ...
	count := ps.sklen()
	sels := make([]SimpleSelector, count)
	for x := 0; x < count; x++ {
		sel := ps.pop().(string)
		sels[count-x-1] = SimpleSelector(sel)
	}
	ps.query.selector = CompoundSelector(sels)
}

func (ps *ParseState) setFunctionSelector() {
	// stack: simple-selector function
	sel := ps.pop().(string)
	op := ps.pop().(string)
	ps.query.selector = &FunctionSelector{op: op, sel: SimpleSelector(sel)}
}

func (ps *ParseState) setNamespace(ns string) {
	ps.query.namespace = ns
}

func (ps *ParseState) setCriteria() {
	// stack: criteria
	ps.query.criteria = ps.pop().(QueryCriteria)
}

func (ps *ParseState) addValueCriteria() {
	// stack: value value-selector ...
	val := ps.pop().(string)
	sel := ps.pop().(string)
	crit := &ValueCriteria{sel: sel, val: val}
	ps.push(crit)
}

func (ps *ParseState) addTimeCriteria() {
	// stack: time op ...
	tstr := ps.pop().(string)
	op := ps.pop().(string)
	ts, err := strconv.Atoi(tstr)
	if err != nil {
		ps.err = err
		ts = 0
	}
	crit := &TimeCriteria{op: op, ts: int64(ts)}
	ps.push(crit)
}

func (ps *ParseState) addCompoundCriteria() {
	// stack: criteria op criteria ...
	right := ps.pop().(QueryCriteria)
	op := ps.pop().(string)
	left := ps.pop().(QueryCriteria)
	crit := &CompoundCriteria{op: op, left: left, right: right}
	ps.push(crit)
}

func (ps *ParseState) setLimit(x string) {
	lim, err := strconv.Atoi(x)
	if err != nil {
		ps.err = err
		lim = 0
	}
	ps.query.limit = lim
}

func (ps *ParseState) push(val interface{}) {
	cell := &ConsCell{car: val, cdr: ps.stack}
	ps.stack = cell
}

func (ps *ParseState) pop() interface{} {
	top := ps.stack.car
	ps.stack = ps.stack.cdr
	return top
}

func (ps *ParseState) sklen() (x int) {
	for next := ps.stack; next != nil; next = next.cdr {
		x++
	}
	return x
}

// query parsing
func ParseQuery(qs string) (*Query, error) {
	ps := &ParseState{query: &Query{}}
	p := &QueryParser{Buffer: qs, ParseState: ps}
	p.Init()
	err := p.Parse()
	if err != nil {
		return nil, err
	}

	p.Execute()
	if ps.err != nil {
		return nil, ps.err
	}

	return ps.query, nil
}

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
type ValueCriteriaFilter func(*pb.Statement, string) bool
type TimeCriteriaFilter func(*pb.Statement, int64) bool
type CompoundCriteriaFilter func(stmt *pb.Statement, left, right StatementFilter) bool

func idCriteriaFilter(stmt *pb.Statement, id string) bool {
	return stmt.Id == id
}

func publisherCriteriaFilter(stmt *pb.Statement, pub string) bool {
	return stmt.Publisher == pub
}

func sourceCriteriaFilter(stmt *pb.Statement, src string) bool {
	// only support simple statements for now, so src = publisher
	return stmt.Publisher == src
}

var valueCriteriaFilters = map[string]ValueCriteriaFilter{
	"id":        idCriteriaFilter,
	"publisher": publisherCriteriaFilter,
	"source":    sourceCriteriaFilter}

func timestampFilterLTEQ(stmt *pb.Statement, ts int64) bool {
	return stmt.Timestamp <= ts
}

func timestampFilterLT(stmt *pb.Statement, ts int64) bool {
	return stmt.Timestamp < ts
}

func timestampFilterEQ(stmt *pb.Statement, ts int64) bool {
	return stmt.Timestamp == ts
}

func timestampFilterGTEQ(stmt *pb.Statement, ts int64) bool {
	return stmt.Timestamp >= ts
}

func timestampFilterGT(stmt *pb.Statement, ts int64) bool {
	return stmt.Timestamp >= ts
}

var timeCriteriaFilters = map[string]TimeCriteriaFilter{
	"<=": timestampFilterLTEQ,
	"<":  timestampFilterLT,
	"=":  timestampFilterEQ,
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
		filter, ok := valueCriteriaFilters[c.sel]
		if !ok {
			return nil, QueryEvalError(fmt.Sprintf("Unexpected criteria selector: %s", c.sel))
		}

		return func(stmt *pb.Statement) bool {
			return filter(stmt, c.val)
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
	"id":        simpleSelectorPublisher,
	"namespace": simpleSelectorNamespace,
	"source":    simpleSelectorSource,
	"timestamp": simpleSelectorTimestamp}

type FunctionStatementSelector func([]interface{}) []interface{}

func countFunctionSelector(res []interface{}) []interface{} {
	return []interface{}{len(res)}
}

var functionSelectors = map[string]FunctionStatementSelector{
	"COUNT": countFunctionSelector}

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

		compf := makeCompoundStatementSelector(sel, getfs)
		return makeSimpleResultSet(compf, query.limit), nil

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

func makeCompoundStatementSelector(sels []SimpleSelector, getfs []StatementSelector) StatementSelector {
	return func(stmt *pb.Statement) interface{} {
		val := make(map[string]interface{})
		for x, sel := range sels {
			val[string(sel)] = getfs[x](stmt)
		}
		return val
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

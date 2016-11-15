package query

import ()

type Query struct {
	Op        int
	namespace string
	selector  QuerySelector
	criteria  QueryCriteria
	order     QueryOrder
	limit     int
}

const (
	OpSelect = iota
	OpDelete
)

func (q *Query) WithLimit(limit int) *Query {
	return &Query{q.Op, q.namespace, q.selector, q.criteria, q.order, limit}
}

func (q *Query) IsSimpleSelect(sel string) bool {
	if q.Op != OpSelect {
		return false
	}

	ssel, ok := q.selector.(SimpleSelector)
	if ok {
		return sel == string(ssel)
	}
	return false
}

func (q *Query) WithSimpleSelect(sel string) *Query {
	return &Query{q.Op, q.namespace, SimpleSelector(sel), q.criteria, q.order, q.limit}
}

type QuerySelector interface {
	selectorType() string
}

type SimpleSelector string
type CompoundSelector []SimpleSelector
type FunctionSelector struct {
	op  string
	sel SimpleSelector
}

func (s SimpleSelector) selectorType() string {
	return "simple"
}

func (s CompoundSelector) selectorType() string {
	return "compound"
}

func (s *FunctionSelector) selectorType() string {
	return "function"
}

type QueryCriteria interface {
	criteriaType() string
}

type ValueCriteria struct {
	op  string
	sel string
	val string
}

type RangeCriteria struct {
	op  string
	sel string
	val int64
}

type IndexCriteria struct {
	sel string
	val string
}

type CompoundCriteria struct {
	op          string
	left, right QueryCriteria
}

type NegatedCriteria struct {
	e QueryCriteria
}

func (c *ValueCriteria) criteriaType() string {
	return "value"
}

func (c *RangeCriteria) criteriaType() string {
	return "range"
}

func (c *IndexCriteria) criteriaType() string {
	return "index"
}

func (c *CompoundCriteria) criteriaType() string {
	return "compound"
}

func (c *NegatedCriteria) criteriaType() string {
	return "negated"
}

type QueryOrder []*QueryOrderSpec

type QueryOrderSpec struct {
	sel string
	dir string
}

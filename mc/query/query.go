package query

import ()

type Query struct {
	Op        int
	namespace string
	selector  QuerySelector
	criteria  QueryCriteria
	limit     int
}

const (
	OpSelect = iota
	OpDelete
)

func (q *Query) WithLimit(limit int) *Query {
	return &Query{q.Op, q.namespace, q.selector, q.criteria, limit}
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
	op  string
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

type NegatedCriteria struct {
	e QueryCriteria
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

func (c *NegatedCriteria) CriteriaType() string {
	return "negated"
}

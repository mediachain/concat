package main

import ()

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
	function string
	selector SimpleSelector
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
	selector string
	value    string
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
}

func (ps *ParseState) setSimpleSelector() {
	// stack: simple-selector
}

func (ps *ParseState) setCompoundSelector() {
	// stack: simple-selector ...
}

func (ps *ParseState) setFunctionSelector() {
	// stack: simple-selector function
}

func (ps *ParseState) setNamespace(ns string) {

}

func (ps *ParseState) setCriteria() {
	// stack: criteria
}

func (ps *ParseState) addValueCriteria() {
	// stack: value value-selector ...
}

func (ps *ParseState) addTimeCriteria() {
	// stack: time comparison-op ...
}

func (ps *ParseState) addCompoundCriteria() {
	// stack: criteria combinator criteria ...
}

func (ps *ParseState) setLimit(x string) {

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

func ParseQuery(qstr string) (*Query, error) {
	ps := &ParseState{query: &Query{}}
	p := &QueryParser{Buffer: qstr, ParseState: ps}
	p.Init()
	err := p.Parse()
	if err != nil {
		return nil, err
	}

	return ps.query, nil
}

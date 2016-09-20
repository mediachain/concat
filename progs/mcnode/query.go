package main

import ()

type Query struct {
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

func (ps *ParseState) addCriteriaCombinator() {
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

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

func (ps *ParseState) setCriteria() {

}

func (ps *ParseState) addValueCriteria() {

}

func (ps *ParseState) addTimeCriteria() {

}

func (ps *ParseState) addCriteriaCombinator() {

}

func (ps *ParseState) push(val interface{}) {

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

package query

import (
	"strconv"
)

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

type ConsCell struct {
	car interface{}
	cdr *ConsCell
}

type ParseState struct {
	query *Query
	stack *ConsCell
	err   error
}

func (ps *ParseState) setSelectOp() {
	ps.query.Op = OpSelect
}

func (ps *ParseState) setDeleteOp() {
	ps.query.Op = OpDelete
	ps.query.selector = SimpleSelector("id")
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
	// stack: value op selector ...
	val := ps.pop().(string)
	op := ps.pop().(string)
	sel := ps.pop().(string)
	crit := &ValueCriteria{op: op, sel: sel, val: val}
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

func (ps *ParseState) addNegatedCriteria() {
	// stack: criteria ...
	e := ps.pop().(QueryCriteria)
	crit := &NegatedCriteria{e}
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

package main

import ()

type Query struct {
}

func ParseQuery(qstr string) (*Query, error) {
	q := &Query{}
	p := &QueryParser{Buffer: qstr, Query: q}
	p.Init()
	err := p.Parse()
	if err != nil {
		return nil, err
	}

	return q, nil
}

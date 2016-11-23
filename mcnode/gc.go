package main

import (
	"context"
	"errors"
)

var (
	NodeMustBeOffline = errors.New("Node must be offline")
)

func (node *Node) doGC(ctx context.Context) (int, error) {
	if node.status != StatusOffline {
		return 0, NodeMustBeOffline
	}

	return 0, nil
}

func (node *Node) doCompact() error {
	if node.status != StatusOffline {
		return NodeMustBeOffline
	}

	node.ds.Compact()
	return nil
}

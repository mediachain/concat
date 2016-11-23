package main

import (
	"context"
)

func (node *Node) doGC(ctx context.Context) (int, error) {
	return 0, nil
}

func (node *Node) doCompact() error {
	return nil
}

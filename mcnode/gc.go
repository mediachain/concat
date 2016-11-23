package main

import (
	"context"
	"database/sql"
	"errors"
)

var (
	NodeMustBeOffline = errors.New("Node must be offline")
)

func (node *Node) doGC(ctx context.Context) (int, error) {
	if node.status != StatusOffline {
		return 0, NodeMustBeOffline
	}

	gc := &GCDB{}
	err := gc.Open(node.home)
	if err != nil {
		return 0, err
	}
	defer gc.Close()

	err = gc.Merge(node.db)
	if err != nil {
		return 0, err
	}

	return gc.GC(node.ds)
}

func (node *Node) doCompact() error {
	if node.status != StatusOffline {
		return NodeMustBeOffline
	}

	node.ds.Compact()
	return nil
}

type GCDB struct {
	db *sql.DB
}

func (gc *GCDB) Open(home string) error {
	return nil
}

func (gc *GCDB) Close() {

}

func (gc *GCDB) Merge(db StatementDB) error {
	return nil
}

func (gc *GCDB) GC(ds Datastore) (int, error) {
	return 0, nil
}

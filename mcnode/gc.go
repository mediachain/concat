package main

import (
	"context"
	"database/sql"
	"errors"
	sqlite3 "github.com/mattn/go-sqlite3"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
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

	err = gc.Merge(ctx, node.db)
	if err != nil {
		return 0, err
	}

	return gc.GC(ctx, node.ds)
}

func (node *Node) doCompact() error {
	if node.status != StatusOffline {
		return NodeMustBeOffline
	}

	node.ds.Compact()
	return nil
}

type GCDB struct {
	db        *sql.DB
	insertKey *sql.Stmt
}

func (gc *GCDB) Open(home string) error {
	// XXX Implement me!
	return nil
}

func (gc *GCDB) Close() {
	// XXX Implement me!
}

func (gc *GCDB) Merge(ctx context.Context, db StatementDB) error {
	q, err := mcq.ParseQuery("SELECT * FROM *")
	if err != nil {
		return err
	}

	ch, err := db.QueryStream(ctx, q)
	if err != nil {
		return err
	}

	const batch = 1024
	keys := make(map[string]bool)

	for val := range ch {
		switch val := val.(type) {
		case *pb.Statement:
			gc.addKeys(val, keys)
			if len(keys) >= batch {
				err := gc.mergeKeys(keys)
				if err != nil {
					return err
				}
				keys = make(map[string]bool)
			}

		case StreamError:
			return val

		default:
			return BadResult
		}
	}

	if len(keys) > 0 {
		return gc.mergeKeys(keys)
	}

	return nil
}

func (gc *GCDB) addKeys(stmt *pb.Statement, keys map[string]bool) error {
	switch body := stmt.Body.Body.(type) {
	case *pb.StatementBody_Simple:
		gc.addSimpleKeys(body.Simple, keys)
		return nil

	case *pb.StatementBody_Compound:
		ss := body.Compound.Body
		for _, s := range ss {
			gc.addSimpleKeys(s, keys)
		}
		return nil

	case *pb.StatementBody_Envelope:
		stmts := body.Envelope.Body
		for _, stmt := range stmts {
			err := gc.addKeys(stmt, keys)
			if err != nil {
				return err
			}
		}
		return nil

	default:
		return BadStatementBody
	}
}

func (gc *GCDB) addSimpleKeys(s *pb.SimpleStatement, keys map[string]bool) {
	keys[s.Object] = true
	for _, dep := range s.Deps {
		keys[dep] = true
	}
}

func (gc *GCDB) mergeKeys(keys map[string]bool) error {
	tx, err := gc.db.Begin()
	if err != nil {
		return err
	}

	insertKey := tx.Stmt(gc.insertKey)

	for key, _ := range keys {
		_, err := insertKey.Exec(key)
		if err != nil {
			xerr, ok := err.(sqlite3.Error)
			if ok && xerr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
				continue
			}
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (gc *GCDB) GC(ctx context.Context, ds Datastore) (int, error) {
	// XXX Implement me!
	return 0, nil
}

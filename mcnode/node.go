package main

import (
	"context"
	"errors"
	"fmt"
	ggproto "github.com/gogo/protobuf/proto"
	p2p_pstore "github.com/ipfs/go-libp2p-peerstore"
	multiaddr "github.com/jbenet/go-multiaddr"
	p2p_host "github.com/libp2p/go-libp2p/p2p/host"
	mc "github.com/mediachain/concat/mc"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type Node struct {
	mc.Identity
	status    int
	laddr     multiaddr.Multiaddr
	host      p2p_host.Host
	dir       *p2p_pstore.PeerInfo
	dirCancel context.CancelFunc
	home      string
	db        StatementDB
	mx        sync.Mutex
	counter   int
}

type StatementDB interface {
	Open(home string) error
	Put(*pb.Statement) error
	Get(id string) (*pb.Statement, error)
	Query(*mcq.Query) ([]interface{}, error)
	QueryStream(context.Context, *mcq.Query) (<-chan interface{}, error)
	Close() error
}

const (
	StatusOffline = iota
	StatusOnline
	StatusPublic
)

var statusString = []string{"offline", "online", "public"}

var UnknownStatement = errors.New("Unknown statement")

func (node *Node) stmtCounter() int {
	node.mx.Lock()
	counter := node.counter
	node.counter++
	node.mx.Unlock()
	return counter
}

var BadStatementBody = errors.New("Unrecognized statement body")

func (node *Node) doPublish(ns string, body interface{}) (string, error) {
	stmt := new(pb.Statement)
	pid := node.ID.Pretty()
	ts := time.Now().Unix()
	counter := node.stmtCounter()
	stmt.Id = fmt.Sprintf("%s:%d:%d", pid, ts, counter)
	stmt.Publisher = pid // this should be the pubkey when we have ECC keys
	stmt.Namespace = ns
	stmt.Timestamp = ts
	switch body := body.(type) {
	case *pb.SimpleStatement:
		stmt.Body = &pb.Statement_Simple{body}

	case *pb.CompoundStatement:
		stmt.Body = &pb.Statement_Compound{body}

	case *pb.EnvelopeStatement:
		stmt.Body = &pb.Statement_Envelope{body}

	case *pb.ArchiveStatement:
		stmt.Body = &pb.Statement_Archive{body}

	default:
		return "", BadStatementBody
	}
	// only sign it with shiny ECC keys, don't bother with RSA
	log.Printf("Publish statement %s", stmt.Id)

	err := node.db.Put(stmt)
	return stmt.Id, err
}

func (node *Node) loadDB() error {
	node.db = &SQLiteDB{}
	return node.db.Open(node.home)
}

// baseline dumb fs/mem db implementation
// Candidate for pruning, but it serves a purpose for now.
type DumbDB struct {
	mx   sync.Mutex
	stmt map[string]*pb.Statement
	dir  string
}

func (db *DumbDB) Put(stmt *pb.Statement) error {
	err := db.saveStatement(stmt)
	if err != nil {
		return err
	}

	db.mx.Lock()
	db.stmt[stmt.Id] = stmt
	db.mx.Unlock()

	return nil
}

func (db *DumbDB) Get(id string) (*pb.Statement, error) {
	db.mx.Lock()
	stmt, ok := db.stmt[id]
	db.mx.Unlock()

	if ok {
		return stmt, nil
	} else {
		return nil, UnknownStatement
	}
}

func (db *DumbDB) Query(q *mcq.Query) ([]interface{}, error) {
	var stmts []*pb.Statement

	db.mx.Lock()
	stmts = make([]*pb.Statement, len(db.stmt))
	x := 0
	for _, stmt := range db.stmt {
		stmts[x] = stmt
		x++
	}
	db.mx.Unlock()

	return mcq.EvalQuery(q, stmts)
}

func (db *DumbDB) QueryStream(ctx context.Context, q *mcq.Query) (<-chan interface{}, error) {
	res, err := db.Query(q)
	if err != nil {
		return nil, err
	}

	ch := make(chan interface{})
	go func() {
		defer close(ch)
		for _, obj := range res {
			select {
			case ch <- obj:
				continue
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func (db *DumbDB) saveStatement(stmt *pb.Statement) error {
	spath := path.Join(db.dir, stmt.Id)

	bytes, err := ggproto.Marshal(stmt)
	if err != nil {
		return err
	}

	log.Printf("Writing statement %s", spath)
	return ioutil.WriteFile(spath, bytes, 0644)
}

func (db *DumbDB) Open(home string) error {
	db.stmt = make(map[string]*pb.Statement)
	db.dir = path.Join(home, "stmt")

	err := os.MkdirAll(db.dir, 0755)
	if err != nil {
		return err
	}

	return db.loadStatements()
}

func (db *DumbDB) loadStatements() error {
	err := filepath.Walk(db.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		log.Printf("Loading statement %s", path)
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		stmt := new(pb.Statement)
		err = ggproto.Unmarshal(bytes, stmt)
		if err != nil {
			return err
		}

		db.stmt[stmt.Id] = stmt
		return nil
	})

	return err
}

func (db *DumbDB) Close() error {
	return nil
}

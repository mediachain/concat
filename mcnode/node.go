package main

import (
	"context"
	"errors"
	"fmt"
	p2p_pstore "github.com/ipfs/go-libp2p-peerstore"
	multiaddr "github.com/jbenet/go-multiaddr"
	p2p_host "github.com/libp2p/go-libp2p/p2p/host"
	mc "github.com/mediachain/concat/mc"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
	"sync"
	"time"
)

type Node struct {
	mc.Identity
	status    int
	laddr     multiaddr.Multiaddr
	host      p2p_host.Host
	netCtx    context.Context
	netCancel context.CancelFunc
	dir       *p2p_pstore.PeerInfo
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
	QueryOne(*mcq.Query) (interface{}, error)
	Delete(*mcq.Query) (int, error)
	Close() error
}

const (
	StatusOffline = iota
	StatusOnline
	StatusPublic
)

var statusString = []string{"offline", "online", "public"}

var (
	UnknownStatement = errors.New("Unknown statement")
	BadStatementBody = errors.New("Unrecognized statement body")
	BadQuery         = errors.New("Unexpected query")
	BadState         = errors.New("Unrecognized state")
	BadMethod        = errors.New("Unsupported method")
	BadNamespace     = errors.New("Illegal namespace")
	NoResult         = errors.New("Empty result set")
)

func (node *Node) stmtCounter() int {
	node.mx.Lock()
	counter := node.counter
	node.counter++
	node.mx.Unlock()
	return counter
}

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
		stmt.Body = &pb.StatementBody{&pb.StatementBody_Simple{body}}

	case *pb.CompoundStatement:
		stmt.Body = &pb.StatementBody{&pb.StatementBody_Compound{body}}

	case *pb.EnvelopeStatement:
		stmt.Body = &pb.StatementBody{&pb.StatementBody_Envelope{body}}

	case *pb.ArchiveStatement:
		stmt.Body = &pb.StatementBody{&pb.StatementBody_Archive{body}}

	default:
		return "", BadStatementBody
	}

	// TODO signatures: only sign it with shiny ECC keys, don't bother with RSA

	err := node.db.Put(stmt)
	return stmt.Id, err
}

func (node *Node) loadDB() error {
	node.db = &SQLiteDB{}
	return node.db.Open(node.home)
}

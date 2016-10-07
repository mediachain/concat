package main

import (
	"context"
	"errors"
	"fmt"
	ggproto "github.com/gogo/protobuf/proto"
	p2p_crypto "github.com/libp2p/go-libp2p-crypto"
	p2p_host "github.com/libp2p/go-libp2p-host"
	p2p_pstore "github.com/libp2p/go-libp2p-peerstore"
	mc "github.com/mediachain/concat/mc"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
	multiaddr "github.com/multiformats/go-multiaddr"
	"sync"
	"time"
)

type Node struct {
	mc.PeerIdentity
	publisher mc.PublisherIdentity
	status    int
	laddr     multiaddr.Multiaddr
	host      p2p_host.Host
	netCtx    context.Context
	netCancel context.CancelFunc
	dir       *p2p_pstore.PeerInfo
	natCfg    NATConfig
	home      string
	db        StatementDB
	mx        sync.Mutex
	counter   int
}

type StatementDB interface {
	Open(home string) error
	Put(*pb.Statement) error
	PutBatch([]*pb.Statement) error
	Get(id string) (*pb.Statement, error)
	Query(*mcq.Query) ([]interface{}, error)
	QueryStream(context.Context, *mcq.Query) (<-chan interface{}, error)
	QueryOne(*mcq.Query) (interface{}, error)
	Merge(*pb.Statement) (bool, error)
	Delete(*mcq.Query) (int, error)
	Close() error
}

const (
	StatusOffline = iota
	StatusOnline
	StatusPublic
)

var statusString = []string{"offline", "online", "public"}

type NATConfig struct {
	opt  int
	addr multiaddr.Multiaddr // public address when option = NATConfigManual
}

const (
	NATConfigNone = iota
	NATConfigAuto
	NATConfigManual
)

var natConfigString = []string{"none", "auto", "manual"}

var (
	UnknownStatement = errors.New("Unknown statement")
	BadStatementBody = errors.New("Unrecognized statement body")
	BadQuery         = errors.New("Unexpected query")
	BadState         = errors.New("Unrecognized state")
	BadMethod        = errors.New("Unsupported method")
	BadNamespace     = errors.New("Illegal namespace")
	BadResult        = errors.New("Bad result set")
	BadStatement     = errors.New("Bad statement; verification failed")
	NoResult         = errors.New("Empty result set")
)

type StreamError struct {
	Err string `json:"error"`
}

func (s StreamError) Error() string {
	return s.Err
}

func sendStreamError(ctx context.Context, ch chan interface{}, what string) {
	select {
	case ch <- StreamError{what}:
	case <-ctx.Done():
	}
}

func (node *Node) stmtCounter() int {
	node.mx.Lock()
	counter := node.counter
	node.counter++
	node.mx.Unlock()
	return counter
}

func (node *Node) doPublish(ns string, body interface{}) (string, error) {
	stmt, err := node.makeStatement(ns, body)
	if err != nil {
		return "", err
	}

	err = node.db.Put(stmt)
	return stmt.Id, err
}

func (node *Node) doPublishBatch(ns string, lst []interface{}) ([]string, error) {
	stmts := make([]*pb.Statement, len(lst))
	sids := make([]string, len(lst))
	for x, body := range lst {
		stmt, err := node.makeStatement(ns, body)
		if err != nil {
			return nil, err
		}
		stmts[x] = stmt
		sids[x] = stmt.Id
	}

	err := node.db.PutBatch(stmts)
	if err != nil {
		return nil, err
	}

	return sids, err
}

func (node *Node) makeStatement(ns string, body interface{}) (*pb.Statement, error) {
	stmt := new(pb.Statement)
	pid := node.publisher.ID58
	ts := time.Now().Unix()
	counter := node.stmtCounter()
	stmt.Id = fmt.Sprintf("%s:%d:%d", pid, ts, counter)
	stmt.Publisher = pid
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
		return nil, BadStatementBody
	}

	err := node.signStatement(stmt)
	if err != nil {
		return nil, err
	}

	return stmt, nil
}

func (node *Node) signStatement(stmt *pb.Statement) error {
	bytes, err := ggproto.Marshal(stmt)
	if err != nil {
		return err
	}

	sig, err := node.publisher.PrivKey.Sign(bytes)
	if err != nil {
		return err
	}

	stmt.Signature = sig
	return nil
}

func (node *Node) verifyStatement(stmt *pb.Statement) (bool, error) {
	pubk, err := mc.PublisherKey(stmt.Publisher)
	if err != nil {
		return false, err
	}

	return node.verifyStatementSig(stmt, pubk)
}

func (node *Node) verifyStatementCacheKeys(stmt *pb.Statement, pkcache map[string]p2p_crypto.PubKey) (bool, error) {
	var pubk p2p_crypto.PubKey
	var err error

	pubk, ok := pkcache[stmt.Publisher]
	if !ok {
		pubk, err = mc.PublisherKey(stmt.Publisher)
		if err != nil {
			return false, err
		}
		pkcache[stmt.Publisher] = pubk
	}

	return node.verifyStatementSig(stmt, pubk)
}

func (node *Node) verifyStatementSig(stmt *pb.Statement, pubk p2p_crypto.PubKey) (bool, error) {
	sig := stmt.Signature
	stmt.Signature = nil
	bytes, err := ggproto.Marshal(stmt)
	stmt.Signature = sig

	if err != nil {
		return false, err
	}

	return pubk.Verify(bytes, sig)
}

func (node *Node) loadDB() error {
	node.db = &SQLiteDB{}
	return node.db.Open(node.home)
}

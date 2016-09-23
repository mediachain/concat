package main

import (
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
	status  int
	laddr   multiaddr.Multiaddr
	host    p2p_host.Host
	dir     *p2p_pstore.PeerInfo
	home    string
	mx      sync.Mutex
	stmt    map[string]*pb.Statement
	counter int
}

const (
	StatusOffline = iota
	StatusOnline
	StatusPublic
)

var statusString = []string{"offline", "online", "public"}

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

	err := node.putStatement(stmt)
	return stmt.Id, err
}

func (node *Node) doQuery(q *mcq.Query) ([]interface{}, error) {
	var stmts []*pb.Statement

	node.mx.Lock()
	stmts = make([]*pb.Statement, len(node.stmt))
	x := 0
	for _, stmt := range node.stmt {
		stmts[x] = stmt
		x++
	}
	node.mx.Unlock()

	return mcq.EvalQuery(q, stmts)
}

func (node *Node) getStatement(id string) (*pb.Statement, bool) {
	node.mx.Lock()
	stmt, ok := node.stmt[id]
	node.mx.Unlock()

	return stmt, ok
}

func (node *Node) putStatement(stmt *pb.Statement) error {
	err := node.saveStatement(stmt)
	if err != nil {
		return err
	}

	node.mx.Lock()
	node.stmt[stmt.Id] = stmt
	node.mx.Unlock()

	return nil
}

func (node *Node) saveStatement(stmt *pb.Statement) error {
	spath := path.Join(node.home, "stmt", stmt.Id)

	bytes, err := ggproto.Marshal(stmt)
	if err != nil {
		return err
	}

	log.Printf("Writing statement %s", spath)
	return ioutil.WriteFile(spath, bytes, 0644)
}

func (node *Node) loadStatements(sdir string) error {
	err := filepath.Walk(sdir, func(path string, info os.FileInfo, err error) error {
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

		node.stmt[stmt.Id] = stmt
		return nil
	})

	return err
}

func (node *Node) loadIndex() error {
	node.stmt = make(map[string]*pb.Statement)

	sdir := path.Join(node.home, "stmt")
	err := os.MkdirAll(sdir, 0755)
	if err != nil {
		return err
	}

	return node.loadStatements(sdir)
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	ggproto "github.com/gogo/protobuf/proto"
	p2p_crypto "github.com/libp2p/go-libp2p-crypto"
	p2p_host "github.com/libp2p/go-libp2p-host"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	p2p_pstore "github.com/libp2p/go-libp2p-peerstore"
	mc "github.com/mediachain/concat/mc"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
	multiaddr "github.com/multiformats/go-multiaddr"
	multihash "github.com/multiformats/go-multihash"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

type Node struct {
	mc.PeerIdentity
	publisher mc.PublisherIdentity
	info      string
	status    int
	laddr     multiaddr.Multiaddr
	host      p2p_host.Host
	netCtx    context.Context
	netCancel context.CancelFunc
	dir       *p2p_pstore.PeerInfo
	natCfg    mc.NATConfig
	home      string
	db        StatementDB
	ds        Datastore
	auth      PeerAuth
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
	MergeBatch([]*pb.Statement) (int, error)
	Delete(*mcq.Query) (int, error)
	Close() error
}

type Key multihash.Multihash
type Datastore interface {
	Open(home string) error
	Put(data []byte) (Key, error)
	PutBatch(batch [][]byte) ([]Key, error)
	Has(Key) (bool, error)
	Get(Key) ([]byte, error)
	Delete(Key) error
	Close()
}

type PeerAuth struct {
	peers map[p2p_peer.ID][]string
	mx    sync.Mutex
}

type NodeInfo struct {
	Peer      string `json:"peer"`
	Publisher string `json:"publisher"`
	Info      string `json:"info"`
}

var (
	UnknownStatement = errors.New("Unknown statement")
	UnknownObject    = errors.New("Unknown Object")
	BadStatementBody = errors.New("Unrecognized statement body")
	BadQuery         = errors.New("Unexpected query")
	BadState         = errors.New("Unrecognized state")
	BadMethod        = errors.New("Unsupported method")
	BadNamespace     = errors.New("Illegal namespace")
	BadResult        = errors.New("Bad result set")
	BadStatement     = errors.New("Bad statement; verification failed")
	NoResult         = errors.New("Empty result set")
	MissingData      = errors.New("Missing statement metadata")
	UnexpectedData   = errors.New("Unexpected data object")
	BadData          = errors.New("Bad data object; hash mismatch")
	BadPush          = errors.New("Bad push value; unexpected object")
	BadResponse      = errors.New("Bad response; unexpected object")
)

const (
	StatusOffline = iota
	StatusOnline
	StatusPublic
)

var statusString = []string{"offline", "online", "public"}

type PushError string

func (s PushError) Error() string {
	return string(s)
}

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

func (node *Node) checkStatement(stmt *pb.Statement) bool {
	return stmt.Id != "" &&
		stmt.Publisher != "" &&
		stmt.Namespace != "" &&
		stmt.Timestamp > 0 &&
		stmt.Body != nil &&
		stmt.Signature != nil
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

func (node *Node) openDB() error {
	node.db = &SQLiteDB{}
	return node.db.Open(node.home)
}

func (node *Node) openDS() error {
	node.ds = &RocksDS{}
	return node.ds.Open(node.home)
}

// persistent configuration
type NodeConfig struct {
	Info string                 `json:"info,omitempty"`
	NAT  string                 `json:"nat,omitempty"`
	Dir  string                 `json:"dir,omitempty"`
	Auth map[string]interface{} `json:"auth,omitempty"`
}

func (node *Node) saveConfig() error {
	var cfg NodeConfig
	cfg.Info = node.info
	cfg.NAT = node.natCfg.String()
	if node.dir != nil {
		cfg.Dir = mc.FormatHandle(*node.dir)
	}
	cfg.Auth = node.auth.toJSON()

	bytes, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	cfgpath := path.Join(node.home, "config.json")
	return ioutil.WriteFile(cfgpath, bytes, 0644)
}

func (node *Node) loadConfig() error {
	cfgpath := path.Join(node.home, "config.json")

	bytes, err := ioutil.ReadFile(cfgpath)
	switch {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return err
	}

	var cfg NodeConfig
	err = json.Unmarshal(bytes, &cfg)
	if err != nil {
		return err
	}

	node.info = cfg.Info

	natCfg, err := mc.NATConfigFromString(cfg.NAT)
	if err != nil {
		return err
	}
	node.natCfg = natCfg

	if cfg.Dir != "" {
		pinfo, err := mc.ParseHandle(cfg.Dir)
		if err != nil {
			return err
		}
		node.dir = &pinfo
	}

	err = node.auth.fromJSON(cfg.Auth)
	return err
}

func (node *Node) doShutdown() {
	log.Printf("Node shutdown")
	err := node.db.Close()
	if err != nil {
		log.Printf("Error closing StatementDB: %s", err.Error())
	}
	node.ds.Close()
	os.Exit(0)
}

func (auth *PeerAuth) fromJSON(map[string]interface{}) error {
	return nil
}

func (auth *PeerAuth) toJSON() map[string]interface{} {
	return nil
}

func (auth *PeerAuth) authorize(pid p2p_peer.ID, nss []string) bool {
	if len(nss) == 0 {
		return false
	}

	auth.mx.Lock()
	defer auth.mx.Unlock()

	rules := auth.peers[pid]
	if len(rules) == 0 {
		return false
	}

	for _, ns := range nss {
		if !auth.authorizeAllow(rules, ns) {
			return false
		}
	}

	return true
}

func (auth *PeerAuth) authorizeAllow(rules []string, ns string) bool {
	for _, rule := range rules {
		switch {
		case rule == "*":
			return true

		case strings.HasSuffix(rule, ".*"):
			if strings.HasPrefix(ns, rule[:len(rule)-2]) {
				return true
			}

		case rule == ns:
			return true
		}
	}

	return false
}

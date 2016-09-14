package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	ggio "github.com/gogo/protobuf/io"
	ggproto "github.com/gogo/protobuf/proto"
	mux "github.com/gorilla/mux"
	p2p_peer "github.com/ipfs/go-libp2p-peer"
	p2p_pstore "github.com/ipfs/go-libp2p-peerstore"
	multiaddr "github.com/jbenet/go-multiaddr"
	p2p_host "github.com/libp2p/go-libp2p/p2p/host"
	p2p_net "github.com/libp2p/go-libp2p/p2p/net"
	mc "github.com/mediachain/concat/mc"
	pb "github.com/mediachain/concat/proto"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type Node struct {
	mc.Identity
	host    p2p_host.Host
	dir     p2p_pstore.PeerInfo
	home    string
	mx      sync.Mutex
	stmt    map[string]*pb.Statement
	counter int
}

func (node *Node) pingHandler(s p2p_net.Stream) {
	defer s.Close()

	pid := s.Conn().RemotePeer()
	log.Printf("node/ping: new stream from %s", pid.Pretty())

	var ping pb.Ping
	var pong pb.Pong
	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	w := ggio.NewDelimitedWriter(s)

	for {
		err := r.ReadMsg(&ping)
		if err != nil {
			return
		}

		log.Printf("node/ping: ping from %s; ponging", pid.Pretty())

		err = w.WriteMsg(&pong)
		if err != nil {
			return
		}
	}
}

func (node *Node) registerPeer(addrs ...multiaddr.Multiaddr) {
	// directory failure is a fatality for now
	ctx := context.Background()

	err := node.host.Connect(ctx, node.dir)
	if err != nil {
		log.Printf("Failed to connect to directory")
		log.Fatal(err)
	}

	s, err := node.host.NewStream(ctx, node.dir.ID, "/mediachain/dir/register")
	if err != nil {
		log.Printf("Failed to open directory stream")
		log.Fatal(err)
	}
	defer s.Close()

	pinfo := p2p_pstore.PeerInfo{node.ID, addrs}
	var pbpi pb.PeerInfo
	mc.PBFromPeerInfo(&pbpi, pinfo)
	msg := pb.RegisterPeer{&pbpi}

	w := ggio.NewDelimitedWriter(s)
	for {
		log.Printf("Registering with directory")
		err = w.WriteMsg(&msg)
		if err != nil {
			log.Printf("Failed to register with directory")
			log.Fatal(err)
		}

		time.Sleep(5 * time.Minute)
	}
}

func (node *Node) httpId(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, node.Identity.Pretty())
}

func (node *Node) httpPing(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	peerId := vars["peerId"]
	pid, err := p2p_peer.IDB58Decode(peerId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error: Bad id: %s\n", err.Error())
		return
	}

	err = node.doPing(r.Context(), pid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Error: %s\n", err.Error())
		return
	}

	fmt.Fprintf(w, "OK\n")
}

func (node *Node) doPing(ctx context.Context, pid p2p_peer.ID) error {
	pinfo, err := node.doLookup(ctx, pid)
	if err != nil {
		return err
	}

	err = node.host.Connect(ctx, pinfo)
	if err != nil {
		return err
	}

	s, err := node.host.NewStream(ctx, pinfo.ID, "/mediachain/node/ping")
	if err != nil {
		return err
	}
	defer s.Close()

	var ping pb.Ping
	w := ggio.NewDelimitedWriter(s)
	err = w.WriteMsg(&ping)
	if err != nil {
		return err
	}

	var pong pb.Pong
	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	err = r.ReadMsg(&pong)

	return err
}

var UnknownPeer = errors.New("Unknown peer")

func (node *Node) doLookup(ctx context.Context, pid p2p_peer.ID) (empty p2p_pstore.PeerInfo, err error) {
	s, err := node.host.NewStream(ctx, node.dir.ID, "/mediachain/dir/lookup")
	if err != nil {
		return empty, err
	}
	defer s.Close()

	req := pb.LookupPeerRequest{string(pid)}
	w := ggio.NewDelimitedWriter(s)
	err = w.WriteMsg(&req)
	if err != nil {
		return empty, err
	}

	var resp pb.LookupPeerResponse
	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	err = r.ReadMsg(&resp)
	if err != nil {
		return empty, err
	}

	if resp.Peer == nil {
		return empty, UnknownPeer
	}

	pinfo, err := mc.PBToPeerInfo(resp.Peer)
	if err != nil {
		return empty, err
	}

	return pinfo, nil
}

func (node *Node) httpPublish(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ns := vars["namespace"]

	rbody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("http/publish: Error reading request body: %s", err.Error())
		return
	}

	// just simple statements for now
	sbody := new(pb.SimpleStatement)
	err = json.Unmarshal(rbody, sbody)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error: %s\n", err.Error())
		return
	}

	sid, err := node.doPublish(ns, sbody)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error: %s\n", err.Error())
		return
	}

	fmt.Fprintln(w, sid)
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
	switch body := body.(type) {
	case *pb.SimpleStatement:
		stmt.Body = &pb.Statement_Simple{body}

	default:
		return "", BadStatementBody
	}
	// only sign it with shiny ECC keys, don't bother with RSA
	log.Printf("Publish statement %s", stmt.Id)

	node.mx.Lock()
	node.stmt[stmt.Id] = stmt
	node.mx.Unlock()

	err := node.saveStatement(stmt)
	if err != nil {
		log.Printf("Warning: Failed to save statement: %s", err.Error())
	}

	return stmt.Id, nil
}

func (node *Node) stmtCounter() int {
	node.mx.Lock()
	counter := node.counter
	node.counter++
	node.mx.Unlock()
	return counter
}

func (node *Node) httpStatement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["statementId"]

	var stmt *pb.Statement
	node.mx.Lock()
	stmt = node.stmt[id]
	node.mx.Unlock()

	if stmt == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "No such statement\n")
		return
	}

	err := json.NewEncoder(w).Encode(stmt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %s\n", err.Error())
		return
	}
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

func (node *Node) loadStatements() {
	sdir := path.Join(node.home, "stmt")
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

	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	pport := flag.Int("l", 9001, "Peer listen port")
	cport := flag.Int("c", 9002, "Peer control interface port [http]")
	home := flag.String("d", "/tmp/mcnode", "Node home")
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options ...] directory\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	addr, err := mc.ParseAddress(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", *pport))
	if err != nil {
		log.Fatal(err)
	}

	dir, err := mc.ParseHandle(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(*home, 0755)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(path.Join(*home, "stmt"), 0755)
	if err != nil {
		log.Fatal(err)
	}

	id, err := mc.NodeIdentity(*home)
	if err != nil {
		log.Fatal(err)
	}

	host, err := mc.NewHost(id, addr)
	if err != nil {
		log.Fatal(err)
	}

	node := &Node{Identity: id, host: host, dir: dir, home: *home, stmt: make(map[string]*pb.Statement)}

	node.loadStatements()

	host.SetStreamHandler("/mediachain/node/ping", node.pingHandler)
	go node.registerPeer(addr)

	log.Printf("I am %s/%s", addr, id.Pretty())

	haddr := fmt.Sprintf("127.0.0.1:%d", *cport)
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/id", node.httpId)
	router.HandleFunc("/ping/{peerId}", node.httpPing)
	router.HandleFunc("/publish/{namespace}", node.httpPublish)
	router.HandleFunc("/stmt/{statementId}", node.httpStatement)

	log.Printf("Serving client interface at %s", haddr)
	err = http.ListenAndServe(haddr, router)
	if err != nil {
		log.Fatal(err)
	}

	select {}
}

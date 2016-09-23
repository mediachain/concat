package main

import (
	"encoding/json"
	"flag"
	"fmt"
	mux "github.com/gorilla/mux"
	p2p_peer "github.com/ipfs/go-libp2p-peer"
	mc "github.com/mediachain/concat/mc"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
)

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
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %s\n", err.Error())
		return
	}

	fmt.Fprintln(w, sid)
}

func (node *Node) httpStatement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["statementId"]

	stmt, ok := node.getStatement(id)
	if !ok {
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

func (node *Node) httpQuery(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("http/query: Error reading request body: %s", err.Error())
		return
	}

	q, err := mcq.ParseQuery(string(body))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error: %s\n", err.Error())
		return
	}

	res, err := node.doQuery(q)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %s\n", err.Error())
		return
	}

	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %s\n", err.Error())
		return
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
	router.HandleFunc("/query", node.httpQuery)

	log.Printf("Serving client interface at %s", haddr)
	err = http.ListenAndServe(haddr, router)
	if err != nil {
		log.Fatal(err)
	}

	select {}
}

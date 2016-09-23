package main

import (
	"flag"
	"fmt"
	mux "github.com/gorilla/mux"
	mc "github.com/mediachain/concat/mc"
	"log"
	"net/http"
	"os"
)

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

	id, err := mc.NodeIdentity(*home)
	if err != nil {
		log.Fatal(err)
	}

	host, err := mc.NewHost(id, addr)
	if err != nil {
		log.Fatal(err)
	}

	node := &Node{Identity: id, host: host, dir: dir, home: *home}

	err = node.loadIndex()
	if err != nil {
		log.Fatal(err)
	}

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

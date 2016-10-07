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

	if len(flag.Args()) != 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options ...]\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	addr, err := mc.ParseAddress(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *pport))
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(*home, 0755)
	if err != nil {
		log.Fatal(err)
	}

	id, err := mc.MakePeerIdentity(*home)
	if err != nil {
		log.Fatal(err)
	}

	pubid, err := mc.MakePublisherIdentity(*home)
	if err != nil {
		log.Fatal(err)
	}

	node := &Node{PeerIdentity: id, publisher: pubid, home: *home, laddr: addr}

	err = node.loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	err = node.loadDB()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Node is offline")

	haddr := fmt.Sprintf("127.0.0.1:%d", *cport)
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/id", node.httpId)
	router.HandleFunc("/ping/{peerId}", node.httpPing)
	router.HandleFunc("/publish/{namespace}", node.httpPublish)
	router.HandleFunc("/stmt/{statementId}", node.httpStatement)
	router.HandleFunc("/query", node.httpQuery)
	router.HandleFunc("/query/{peerId}", node.httpRemoteQuery)
	router.HandleFunc("/merge/{peerId}", node.httpMerge)
	router.HandleFunc("/delete", node.httpDelete)
	router.HandleFunc("/status", node.httpStatus)
	router.HandleFunc("/status/{state}", node.httpStatusSet)
	router.HandleFunc("/config/dir", node.httpConfigDir)
	router.HandleFunc("/config/nat", node.httpConfigNAT)

	log.Printf("Serving client interface at %s", haddr)
	err = http.ListenAndServe(haddr, router)
	if err != nil {
		log.Fatal(err)
	}

	select {}
}

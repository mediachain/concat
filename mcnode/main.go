package main

import (
	"flag"
	"fmt"
	mux "github.com/gorilla/mux"
	mc "github.com/mediachain/concat/mc"
	homedir "github.com/mitchellh/go-homedir"
	"log"
	"net/http"
	"os"
)

func main() {
	pport := flag.Int("l", 9001, "Peer listen port")
	cport := flag.Int("c", 9002, "Peer control interface port [http]")
	bindaddr := flag.String("b", "127.0.0.1", "Peer control bind address [http]")
	hdir := flag.String("d", "~/.mediachain/mcnode", "Node home")
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

	home, err := homedir.Expand(*hdir)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(home, 0755)
	if err != nil {
		log.Fatal(err)
	}

	id, err := mc.MakePeerIdentity(home)
	if err != nil {
		log.Fatal(err)
	}

	pubid, err := mc.MakePublisherIdentity(home)
	if err != nil {
		log.Fatal(err)
	}

	node := &Node{PeerIdentity: id, publisher: pubid, home: home, laddr: addr}

	err = node.loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	err = node.openDB()
	if err != nil {
		log.Fatal(err)
	}

	err = node.openDS()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Node is offline")

	haddr := fmt.Sprintf("%s:%d", *bindaddr, *cport)
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/id", node.httpId)
	router.HandleFunc("/id/{peerId}", node.httpRemoteId)
	router.HandleFunc("/ping/{peerId}", node.httpPing)
	router.HandleFunc("/publish/{namespace}", node.httpPublish)
	router.HandleFunc("/publish/{namespace}/{combine}", node.httpPublishCompound)
	router.HandleFunc("/stmt/{statementId}", node.httpStatement)
	router.HandleFunc("/query", node.httpQuery)
	router.HandleFunc("/query/{peerId}", node.httpRemoteQuery)
	router.HandleFunc("/merge/{peerId}", node.httpMerge)
	router.HandleFunc("/push/{peerId}", node.httpPush)
	router.HandleFunc("/delete", node.httpDelete)
	router.HandleFunc("/data/put", node.httpPutData)
	router.HandleFunc("/data/get/{objectId}", node.httpGetData)
	router.HandleFunc("/data/merge/{peerId}", node.httpMergeData)
	router.HandleFunc("/data/keys", node.httpDataKeys)
	router.HandleFunc("/data/gc", node.httpGCData)
	router.HandleFunc("/status", node.httpStatus)
	router.HandleFunc("/status/{state}", node.httpStatusSet)
	router.HandleFunc("/config/dir", node.httpConfigDir)
	router.HandleFunc("/config/nat", node.httpConfigNAT)
	router.HandleFunc("/config/info", node.httpConfigInfo)
	router.HandleFunc("/auth", node.httpAuth)
	router.HandleFunc("/auth/{peerId}", node.httpAuthPeer)
	router.HandleFunc("/dir/list", node.httpDirList)
	router.HandleFunc("/net/addr", node.httpNetAddr)
	router.HandleFunc("/shutdown", node.httpShutdown)

	log.Printf("Serving client interface at %s", haddr)
	err = http.ListenAndServe(haddr, router)
	if err != nil {
		log.Fatal(err)
	}

	select {}
}

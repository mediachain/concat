package main

import (
	"encoding/json"
	"fmt"
	mux "github.com/gorilla/mux"
	p2p_peer "github.com/ipfs/go-libp2p-peer"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
	"io/ioutil"
	"log"
	"net/http"
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

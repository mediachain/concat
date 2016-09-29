package main

import (
	"context"
	"encoding/json"
	"fmt"
	mux "github.com/gorilla/mux"
	p2p_peer "github.com/ipfs/go-libp2p-peer"
	mc "github.com/mediachain/concat/mc"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func apiError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	fmt.Fprintf(w, "Error: %s\n", err.Error())
}

func (node *Node) httpId(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, node.Identity.Pretty())
}

func (node *Node) httpPing(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	peerId := vars["peerId"]
	pid, err := p2p_peer.IDB58Decode(peerId)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	err = node.doPing(r.Context(), pid)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}

	fmt.Fprintln(w, "OK")
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
		apiError(w, http.StatusBadRequest, err)
		return
	}

	sid, err := node.doPublish(ns, sbody)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}

	fmt.Fprintln(w, sid)
}

func (node *Node) httpStatement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["statementId"]

	stmt, err := node.db.Get(id)
	if err != nil {
		switch err {
		case UnknownStatement:
			apiError(w, http.StatusNotFound, err)
			return

		default:
			apiError(w, http.StatusInternalServerError, err)
			return
		}
	}

	err = json.NewEncoder(w).Encode(stmt)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
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
		apiError(w, http.StatusBadRequest, err)
		return
	}

	if q.Op != mcq.OpSelect {
		apiError(w, http.StatusBadRequest, BadQuery)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	ch, err := node.db.QueryStream(ctx, q)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}

	enc := json.NewEncoder(w)
	for obj := range ch {
		err = enc.Encode(obj)
		if err != nil {
			log.Printf("Error encoding query result: %s", err.Error())
			return
		}
	}
}

func (node *Node) httpDelete(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("http/query: Error reading request body: %s", err.Error())
		return
	}

	q, err := mcq.ParseQuery(string(body))
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	if q.Op != mcq.OpDelete {
		apiError(w, http.StatusBadRequest, BadQuery)
		return
	}

	count, err := node.db.Delete(q)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}

	fmt.Fprintln(w, count)
}

func (node *Node) httpStatus(w http.ResponseWriter, r *http.Request) {
	status := statusString[node.status]
	fmt.Fprintln(w, status)
}

func (node *Node) httpStatusSet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	state := vars["state"]

	var err error
	switch state {
	case "offline":
		err = node.goOffline()

	case "online":
		err = node.goOnline()

	case "public":
		err = node.goPublic()

	default:
		apiError(w, http.StatusBadRequest, BadState)
		return
	}

	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}

	fmt.Fprintln(w, statusString[node.status])
}

func (node *Node) httpConfigDir(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodHead:
		return
	case http.MethodGet:
		if node.dir != nil {
			fmt.Fprintln(w, mc.FormatHandle(*node.dir))
		} else {
			fmt.Fprintln(w, "nil")
		}
	case http.MethodPost:
		node.httpConfigDirSet(w, r)

	default:
		apiError(w, http.StatusBadRequest, BadMethod)
	}
}

func (node *Node) httpConfigDirSet(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("http/query: Error reading request body: %s", err.Error())
		return
	}

	handle := strings.TrimSpace(string(body))
	pinfo, err := mc.ParseHandle(handle)

	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	node.dir = &pinfo
	fmt.Fprintln(w, "OK")
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	mux "github.com/gorilla/mux"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	mc "github.com/mediachain/concat/mc"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func apiError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	fmt.Fprintf(w, "Error: %s\n", err.Error())
}

// Local node REST API implementation

// GET /id
// Returns the node peer identity.
func (node *Node) httpId(w http.ResponseWriter, r *http.Request) {
	nids := NodeIds{node.NodeIdentity.Pretty(), node.publisher.ID58}

	err := json.NewEncoder(w).Encode(nids)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
}

type NodeIds struct {
	Peer      string `json:"peer"`
	Publisher string `json:"publisher"`
}

// GET /ping/{peerId}
// Lookup a peer in the directory and ping it with the /mediachain/node/ping protocol.
// The node must be online and a directory must have been configured.
func (node *Node) httpPing(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	peerId := vars["peerId"]
	pid, err := p2p_peer.IDB58Decode(peerId)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	err = node.doPing(ctx, pid)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}

	fmt.Fprintln(w, "OK")
}

var nsrx *regexp.Regexp

func init() {
	rx, err := regexp.Compile("^[a-zA-Z0-9-]+([.][a-zA-Z0-9-]+)*$")
	if err != nil {
		log.Fatal(err)
	}
	nsrx = rx
}

// POST /publish/{namespace}
// DATA: A stream of json-encoded pb.SimpleStatements
// Publishes a batch of statements to the specified namespace.
// Returns the statement ids as a newline delimited stream.
func (node *Node) httpPublish(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ns := vars["namespace"]

	rbody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("http/publish: Error reading request body: %s", err.Error())
		return
	}

	if !nsrx.Match([]byte(ns)) {
		apiError(w, http.StatusBadRequest, BadNamespace)
		return
	}

	dec := json.NewDecoder(bytes.NewReader(rbody))
	stmts := make([]interface{}, 0)

loop:
	for {
		sbody := new(pb.SimpleStatement)
		err = dec.Decode(sbody)
		switch {
		case err == io.EOF:
			break loop
		case err != nil:
			apiError(w, http.StatusBadRequest, err)
			return
		default:
			stmts = append(stmts, sbody)
		}
	}

	if len(stmts) == 0 {
		return
	}

	sids, err := node.doPublishBatch(ns, stmts)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}

	for _, sid := range sids {
		fmt.Fprintln(w, sid)
	}
}

// GET /stmt/{statementId}
// Retrieves a statement by id
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

// POST /query
// DATA: MCQL SELECT query
// Queries the statement database and return the result set in ndjson
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

// POST /query/{peerId}
// DATA: MCQL SELECT query
// Queries a remote peer and returns the result set in ndjson
func (node *Node) httpRemoteQuery(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	peerId := vars["peerId"]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("http/query: Error reading request body: %s", err.Error())
		return
	}

	q := string(body)

	qq, err := mcq.ParseQuery(q)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	if qq.Op != mcq.OpSelect {
		apiError(w, http.StatusBadRequest, BadQuery)
		return
	}

	pid, err := p2p_peer.IDB58Decode(peerId)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	ch, err := node.doRemoteQuery(ctx, pid, q)
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

// POST /merge/{peerId}
// DATA: MCQL SELECT query
// Queries a remote peer and merges the resulting statements into the local
// db; returns the number of statements merged
func (node *Node) httpMerge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	peerId := vars["peerId"]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("http/merge: Error reading request body: %s", err.Error())
		return
	}

	q := string(body)

	qq, err := mcq.ParseQuery(q)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	if !qq.IsSimpleSelect("*") {
		apiError(w, http.StatusBadRequest, BadQuery)
		return
	}

	pid, err := p2p_peer.IDB58Decode(peerId)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	count, err := node.doMerge(ctx, pid, q)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		if count > 0 {
			fmt.Fprintf(w, "Partial merge: %d statements merged\n", count)
		}
		return
	}

	fmt.Fprintln(w, count)
}

// POST /delete
// DATA: MCQL DELTE query
// Deletes statements from the statement db
// Returns the number of statements deleted
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
		if count > 0 {
			fmt.Fprintf(w, "Partial delete: %d statements deleted\n", count)
		}
		return
	}

	fmt.Fprintln(w, count)
}

// GET /status
// Returns the node network state
func (node *Node) httpStatus(w http.ResponseWriter, r *http.Request) {
	status := statusString[node.status]
	fmt.Fprintln(w, status)
}

// POST /status/{state}
// Effects the network state
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

// GET  /config/dir
// POST /config/dir
// retrieve/set the configured directory
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

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	mux "github.com/gorilla/mux"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	mc "github.com/mediachain/concat/mc"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
	multihash "github.com/multiformats/go-multihash"
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
// Returns the node info, which includes the peer and publisher ids, and the
// configured node information.
func (node *Node) httpId(w http.ResponseWriter, r *http.Request) {
	ninfo := NodeInfo{node.PeerIdentity.Pretty(), node.publisher.Pretty(), node.info}

	err := json.NewEncoder(w).Encode(ninfo)
	if err != nil {
		log.Printf("Error writing response body: %s", err.Error())
	}
}

// GET /id/{peerId}
// Returns the node info from a remote peer
func (node *Node) httpRemoteId(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	peerId := vars["peerId"]
	pid, err := p2p_peer.IDB58Decode(peerId)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	ninfo, err := node.doRemoteId(ctx, pid)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}

	err = json.NewEncoder(w).Encode(ninfo)
	if err != nil {
		log.Printf("Error writing response body: %s", err.Error())
	}
}

// GET /net/addr
// Returns all routable node addrs
func (node *Node) httpNetAddr(w http.ResponseWriter, r *http.Request) {
	addrs := node.netAddrs()
	for _, addr := range addrs {
		fmt.Fprintln(w, addr.String())
	}
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

// GET /dir/list
// List peers known to the directory
func (node *Node) httpDirList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	peers, err := node.doDirList(ctx)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}

	// filter self from result set
	mypid := node.PeerIdentity.Pretty()
	for _, peer := range peers {
		if peer != mypid {
			fmt.Fprintln(w, peer)
		}
	}
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

	if !nsrx.Match([]byte(ns)) {
		apiError(w, http.StatusBadRequest, BadNamespace)
		return
	}

	dec := json.NewDecoder(r.Body)
	stmts := make([]interface{}, 0, 1024)

loop:
	for {
		sbody := new(pb.SimpleStatement)
		err := dec.Decode(sbody)
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
		log.Printf("Error writing response body: %s", err.Error())
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

	count, ocount, err := node.doMerge(ctx, pid, q)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		if count > 0 {
			fmt.Fprintf(w, "Partial merge: %d statements merged\n", count)
		}
		if ocount > 0 {
			fmt.Fprintf(w, "Partial merge: %d objects merged\n", ocount)
		}

		return
	}

	fmt.Fprintln(w, count)
	fmt.Fprintln(w, ocount)
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

// datastore interface
type DataObject struct {
	Data []byte `json:"data"`
}

// Get /data/get/{objectId}
// Retrieves a data object from the datastore
func (node *Node) httpGetData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key58 := vars["objectId"]
	key, err := multihash.FromB58String(key58)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	data, err := node.ds.Get(Key(key))
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}

	if data == nil {
		apiError(w, http.StatusNotFound, UnknownObject)
		return
	}

	dao := DataObject{data}
	err = json.NewEncoder(w).Encode(dao)
	if err != nil {
		log.Printf("Error writing response body: %s", err.Error())
	}
}

// POST /data/put
// DATA: A stream of json-encoded data objects
// Puts a batch of objects to the datastore
// returns a stream of object ids (B58 encoded content multihashes)
func (node *Node) httpPutData(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	batch := make([][]byte, 0, 1024)

	var dao DataObject
loop:
	for {
		err := dec.Decode(&dao)
		switch {
		case err == io.EOF:
			break loop
		case err != nil:
			apiError(w, http.StatusBadRequest, err)
			return
		default:
			batch = append(batch, dao.Data)
			dao.Data = nil
		}
	}

	// We don't use the PutBatch interface, as it performs a synchronous write
	// of the batch, which requires us to carefully tune batch sizes in order
	// to achieve comparable performance with async writes.
	// But we do want a single error response from the API, without forcing the
	// client to parse a stream.
	// So the batch is written individually with asynchronous writes, and produces
	// a single error result or a stream of hashes.
	// Writes are idempotent, so there is no deleterious effect from partial writes
	// (other than a subset of the objects written in the datastore)
	keys := make([]Key, len(batch))
	for x, data := range batch {
		key, err := node.ds.Put(data)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		keys[x] = key
	}

	for _, key := range keys {
		fmt.Fprintln(w, multihash.Multihash(key).B58String())
	}
}

// Get /data/merge/{peerId}
// Merges raw data objects from peerId
func (node *Node) httpMergeData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	peerId := vars["peerId"]

	pid, err := p2p_peer.IDB58Decode(peerId)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	keys := make(map[string]Key)

	scanner := bufio.NewScanner(r.Body)
	for scanner.Scan() {
		err = node.mergeObjectKey(scanner.Text(), keys)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	}

	count, err := node.doRawMerge(r.Context(), pid, keys)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		if count > 0 {
			fmt.Fprintf(w, "Partial merge: %d objects merged\n", count)
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

// config api
func apiConfigMethod(w http.ResponseWriter, r *http.Request, getf, setf http.HandlerFunc) {
	switch r.Method {
	case http.MethodHead:
		return
	case http.MethodGet:
		getf(w, r)
	case http.MethodPost:
		setf(w, r)

	default:
		apiError(w, http.StatusBadRequest, BadMethod)
	}
}

// GET  /config/dir
// POST /config/dir
// retrieve/set the configured directory
func (node *Node) httpConfigDir(w http.ResponseWriter, r *http.Request) {
	apiConfigMethod(w, r, node.httpConfigDirGet, node.httpConfigDirSet)
}

func (node *Node) httpConfigDirGet(w http.ResponseWriter, r *http.Request) {
	if node.dir != nil {
		fmt.Fprintln(w, mc.FormatHandle(*node.dir))
	} else {
		fmt.Fprintln(w, "nil")
	}
}

func (node *Node) httpConfigDirSet(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("http/config/dir: Error reading request body: %s", err.Error())
		return
	}

	handle := strings.TrimSpace(string(body))
	pinfo, err := mc.ParseHandle(handle)

	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	node.dir = &pinfo

	err = node.saveConfig()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}

	fmt.Fprintln(w, "OK")
}

// GET  /config/nat
// POST /config/nat
// retrieve/set the NAT configuration
func (node *Node) httpConfigNAT(w http.ResponseWriter, r *http.Request) {
	apiConfigMethod(w, r, node.httpConfigNATGet, node.httpConfigNATSet)
}

func (node *Node) httpConfigNATGet(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, node.natCfg.String())
}

func (node *Node) httpConfigNATSet(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("http/config/nat: Error reading request body: %s", err.Error())
		return
	}

	opt := strings.TrimSpace(string(body))
	cfg, err := mc.NATConfigFromString(opt)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	node.natCfg = cfg

	err = node.saveConfig()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}

	fmt.Fprintln(w, "OK")
}

// GET  /config/info
// POST /config/info
// retrieve/set node information
func (node *Node) httpConfigInfo(w http.ResponseWriter, r *http.Request) {
	apiConfigMethod(w, r, node.httpConfigInfoGet, node.httpConfigInfoSet)
}

func (node *Node) httpConfigInfoGet(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, node.info)
}

func (node *Node) httpConfigInfoSet(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("http/config/info: Error reading request body: %s", err.Error())
		return
	}

	node.info = strings.TrimSpace(string(body))

	err = node.saveConfig()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}

	fmt.Fprintln(w, "OK")
}

// POST /shutdown
// shutdown the node
func (node *Node) httpShutdown(w http.ResponseWriter, r *http.Request) {
	node.doShutdown()
}

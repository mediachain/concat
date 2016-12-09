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
	"strconv"
	"strings"
	"time"
)

func apiError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	fmt.Fprintf(w, "Error: %s\n", err.Error())
}

func apiNetError(w http.ResponseWriter, err error) {
	switch err {
	case UnknownPeer:
		apiError(w, http.StatusNotFound, err)
	default:
		apiError(w, http.StatusInternalServerError, err)
	}
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
		apiNetError(w, err)
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

// GET /net/addr/{peerId}
// Returns all peer addrs in the local cache
func (node *Node) httpNetPeerAddr(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	peerId := vars["peerId"]
	pid, err := p2p_peer.IDB58Decode(peerId)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	addrs := node.netPeerAddrs(pid)
	for _, addr := range addrs {
		fmt.Fprintln(w, addr.String())
	}
}

// GET /net/conns
// Returns all active peer connections
func (node *Node) httpNetConns(w http.ResponseWriter, r *http.Request) {
	peers := node.netConns()
	for _, peer := range peers {
		fmt.Fprintln(w, mc.FormatHandle(peer))
	}
}

// GET /net/lookup/{peerId}
// Looks up a peer in the network
func (node *Node) httpNetLookup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	peerId := vars["peerId"]
	pid, err := p2p_peer.IDB58Decode(peerId)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	pinfo, err := node.doLookup(ctx, pid)
	if err != nil {
		apiNetError(w, err)
		return
	}

	for _, addr := range pinfo.Addrs {
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
		apiNetError(w, err)
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

// POST /publish/{namespace}/{combine}
// DATA: A stream of json-encoded pb.SimpleStatements using CompoundStatement grouping
// Publishes a batch of statements to the specified namespace.
// Returns the statement ids as a newline delimited stream.
func (node *Node) httpPublishCompound(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	ns := vars["namespace"]
	if !nsrx.Match([]byte(ns)) {
		apiError(w, http.StatusBadRequest, BadNamespace)
		return
	}

	comb := vars["combine"]
	clen, err := strconv.Atoi(comb)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	dec := json.NewDecoder(r.Body)
	stmts := make([]interface{}, 0, 1000/clen)
	body := make([]*pb.SimpleStatement, 0, clen)

loop:
	for {
		for x := 0; x < clen; x++ {
			sbody := new(pb.SimpleStatement)
			err = dec.Decode(sbody)
			switch {
			case err == io.EOF:
				break loop
			case err != nil:
				apiError(w, http.StatusBadRequest, err)
				return
			default:
				body = append(body, sbody)
			}
		}

		stmt := &pb.CompoundStatement{body}
		stmts = append(stmts, stmt)
		body = make([]*pb.SimpleStatement, 0, clen)
	}

	if len(body) > 0 {
		stmt := &pb.CompoundStatement{body}
		stmts = append(stmts, stmt)
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
		apiNetError(w, err)
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
// db; returns the number of statements and objects merged
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
		apiNetError(w, err)
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

// POST /push/{peerId}
// DATA: MCQL SELECT query
// Pushes statements matching the query to peerId for merge; must be
// authorized for push by the peer
// returns the number of statements and objects merged
func (node *Node) httpPush(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	peerId := vars["peerId"]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("http/push: Error reading request body: %s", err.Error())
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

	count, ocount, err := node.doPush(ctx, pid, qq)
	if err != nil {
		apiNetError(w, err)
		if count > 0 {
			fmt.Fprintf(w, "Partial push: %d statements merged\n", count)
		}
		if ocount > 0 {
			fmt.Fprintf(w, "Partial push: %d objects merged\n", ocount)
		}
		if count < 0 {
			fmt.Fprintf(w, "Incomplete push: some statements may have been merged\n")
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

// GET /data/get/{objectId}
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

// POST /data/get
// Retrieves a batch of data objects from the datastore
func (node *Node) httpGetDataBatch(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	scanner := bufio.NewScanner(r.Body)
	for scanner.Scan() {
		key58 := scanner.Text()

		key, err := multihash.FromB58String(key58)
		if err != nil {
			enc.Encode(StreamError{err.Error()})
			return
		}

		data, err := node.ds.Get(Key(key))
		if err != nil {
			enc.Encode(StreamError{err.Error()})
			return
		}

		err = enc.Encode(DataObject{data})
		if err != nil {
			log.Printf("Error writing response body: %s", err.Error())
			return
		}
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

// POST /data/merge/{peerId}
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
		apiNetError(w, err)
		if count > 0 {
			fmt.Fprintf(w, "Partial merge: %d objects merged\n", count)
		}
		return
	}

	fmt.Fprintln(w, count)
}

// POST /data/gc
// garbage collect orphan data objects that are not referenced by any statement
func (node *Node) httpGCData(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	count, err := node.doGC(ctx)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		if count > 0 {
			fmt.Fprintf(w, "Partial GC: %d objects deleted\n", count)
		}
		return
	}

	fmt.Fprintln(w, count)
}

// POST /data/compact
// compact the datastore
func (node *Node) httpCompactData(w http.ResponseWriter, r *http.Request) {
	err := node.doCompact()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}

	fmt.Fprintln(w, "OK")
}

// POST /data/sync
// flushes the datastore; useful for immediately reclaiming space used by the WAL
func (node *Node) httpSyncData(w http.ResponseWriter, r *http.Request) {
	err := node.ds.Sync()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}

	fmt.Fprintln(w, "OK")
}

func (node *Node) httpDataKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := node.ds.IterKeys(r.Context())
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}

	for key := range keys {
		fmt.Fprintln(w, multihash.Multihash(key).B58String())
	}
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

// GET /auth
// retrieves all peer authorization rules in json
func (node *Node) httpAuth(w http.ResponseWriter, r *http.Request) {
	rules := node.auth.toJSON()

	err := json.NewEncoder(w).Encode(rules)
	if err != nil {
		log.Printf("Error writing response body: %s", err.Error())
	}
}

// GET  /auth/{peerId}
// POST /auth/{peerId}
// gets/sets auth rules for peerId
// rules are specified as a comma separated list of namespaces (or ns wildcards)
func (node *Node) httpAuthPeer(w http.ResponseWriter, r *http.Request) {
	apiConfigMethod(w, r, node.httpAuthPeerGet, node.httpAuthPeerSet)
}

func (node *Node) httpAuthPeerGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	peerId := vars["peerId"]

	pid, err := p2p_peer.IDB58Decode(peerId)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	rules := node.auth.getRules(pid)
	if len(rules) > 0 {
		fmt.Fprintln(w, strings.Join(rules, ","))
	}
}

func (node *Node) httpAuthPeerSet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	peerId := vars["peerId"]

	pid, err := p2p_peer.IDB58Decode(peerId)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("http/auth: Error reading request body: %s", err.Error())
		return
	}

	rbody := strings.TrimSpace(string(body))
	if rbody == "" {
		node.auth.clearRules(pid)
	} else {
		rules := strings.Split(rbody, ",")
		node.auth.setRules(pid, rules)
	}

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

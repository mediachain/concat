package main

import (
	"bytes"
	"context"
	"errors"
	ggio "github.com/gogo/protobuf/io"
	p2p_crypto "github.com/libp2p/go-libp2p-crypto"
	p2p_net "github.com/libp2p/go-libp2p-net"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	p2p_pstore "github.com/libp2p/go-libp2p-peerstore"
	mc "github.com/mediachain/concat/mc"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
	multiaddr "github.com/multiformats/go-multiaddr"
	multihash "github.com/multiformats/go-multihash"
	"log"
	"runtime"
	"time"
)

var (
	NodeOffline  = errors.New("Node is offline")
	NoDirectory  = errors.New("No directory server")
	UnknownPeer  = errors.New("Unknown peer")
	IllegalState = errors.New("Illegal node state")
)

// goOffline stops the network
func (node *Node) goOffline() error {
	node.mx.Lock()
	defer node.mx.Unlock()

	switch node.status {
	case StatusPublic:
		fallthrough
	case StatusOnline:
		node.netCancel()
		err := node.host.Close()
		if err != nil {
			log.Printf("Error closing host: %s", err.Error())
		}
		node.status = StatusOffline
		node.natCfg.Clear()
		log.Println("Node is offline")
		return err

	default:
		return nil
	}
}

// goOnline starts the network; if the node is already public, it stays public
func (node *Node) goOnline() error {
	node.mx.Lock()
	defer node.mx.Unlock()

	switch node.status {
	case StatusOffline:
		err := node._goOnline()
		if err != nil {
			return err
		}

		node.status = StatusOnline
		log.Println("Node is online")
		return nil

	default:
		return nil
	}
}

func (node *Node) _goOnline() error {
	var opts []interface{}
	if node.natCfg.Opt == mc.NATConfigAuto {
		opts = []interface{}{mc.NATPortMap}
	}

	host, err := mc.NewHost(node.PeerIdentity, node.laddr, opts...)
	if err != nil {
		return err
	}

	host.SetStreamHandler("/mediachain/node/id", node.idHandler)
	host.SetStreamHandler("/mediachain/node/ping", node.pingHandler)
	host.SetStreamHandler("/mediachain/node/query", node.queryHandler)
	host.SetStreamHandler("/mediachain/node/data", node.dataHandler)
	node.host = host

	ctx, cancel := context.WithCancel(context.Background())
	node.netCtx = ctx
	node.netCancel = cancel

	return nil
}

// goPublic starts the network if it's not already up and registers with the
// directory; fails with NoDirectory if that hasn't been configured.
func (node *Node) goPublic() error {
	if node.dir == nil {
		return NoDirectory
	}

	node.mx.Lock()
	defer node.mx.Unlock()

	switch node.status {
	case StatusOffline:
		err := node._goOnline()
		if err != nil {
			return err
		}
		fallthrough

	case StatusOnline:
		go node.registerPeer(node.netCtx)
		node.status = StatusPublic

		log.Println("Node is public")
		return nil

	default:
		return nil
	}
}

func (node *Node) pingHandler(s p2p_net.Stream) {
	defer s.Close()

	pid := s.Conn().RemotePeer()
	paddr := s.Conn().RemoteMultiaddr()
	log.Printf("node/ping: new stream from %s at %s", pid.Pretty(), paddr.String())

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

func (node *Node) idHandler(s p2p_net.Stream) {
	defer s.Close()

	pid := s.Conn().RemotePeer()
	paddr := s.Conn().RemoteMultiaddr()
	log.Printf("node/id: new stream from %s at %s", pid.Pretty(), paddr.String())

	var req pb.NodeInfoRequest
	var res pb.NodeInfo
	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	w := ggio.NewDelimitedWriter(s)

	err := r.ReadMsg(&req)
	if err != nil {
		return
	}

	res.Peer = node.PeerIdentity.Pretty()
	res.Publisher = node.publisher.ID58
	res.Info = node.info

	w.WriteMsg(&res)
}

func (node *Node) queryHandler(s p2p_net.Stream) {
	defer s.Close()
	pid := s.Conn().RemotePeer()
	paddr := s.Conn().RemoteMultiaddr()
	log.Printf("node/query: new stream from %s at %s", pid.Pretty(), paddr.String())

	ctx, cancel := context.WithCancel(node.netCtx)
	defer cancel()

	var req pb.QueryRequest
	var res pb.QueryResult

	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	w := ggio.NewDelimitedWriter(s)

	writeError := func(err error) {
		res.Result = &pb.QueryResult_Error{&pb.StreamError{err.Error()}}
		w.WriteMsg(&res)
	}

	writeEnd := func() error {
		res.Result = &pb.QueryResult_End{&pb.StreamEnd{}}
		return w.WriteMsg(&res)
	}

	writeValue := func(val interface{}) error {
		switch val := val.(type) {
		case map[string]interface{}:
			cv, err := mc.CompoundValue(val)
			if err != nil {
				log.Printf("node/query: %s", err.Error())
				writeError(err)
				return err
			}

			res.Result = &pb.QueryResult_Value{&pb.QueryResultValue{
				&pb.QueryResultValue_Compound{cv}}}

		default:
			sv, err := mc.SimpleValue(val)
			if err != nil {
				log.Printf("node/query: %s", err.Error())
				writeError(err)
				return err
			}

			res.Result = &pb.QueryResult_Value{&pb.QueryResultValue{
				&pb.QueryResultValue_Simple{sv}}}
		}

		return w.WriteMsg(&res)
	}

	for {
		err := r.ReadMsg(&req)
		if err != nil {
			return
		}

		log.Printf("node/query: query from %s: %s", pid.Pretty(), req.Query)

		q, err := mcq.ParseQuery(req.Query)
		if err != nil {
			writeError(err)
			return
		}

		if q.Op != mcq.OpSelect {
			writeError(BadQuery)
			return
		}

		ch, err := node.db.QueryStream(ctx, q)
		if err != nil {
			writeError(err)
			return
		}

		for val := range ch {
			err = writeValue(val)
			if err != nil {
				return
			}
		}

		err = writeEnd()
		if err != nil {
			return
		}

		req.Reset()
	}
}

func (node *Node) dataHandler(s p2p_net.Stream) {
	defer s.Close()
	pid := s.Conn().RemotePeer()
	paddr := s.Conn().RemoteMultiaddr()
	log.Printf("node/data: new stream from %s at %s", pid.Pretty(), paddr.String())

	var req pb.DataRequest
	var res pb.DataResult

	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	w := ggio.NewDelimitedWriter(s)

	writeError := func(err error) {
		res.Result = &pb.DataResult_Error{&pb.StreamError{err.Error()}}
		w.WriteMsg(&res)
	}

	writeEnd := func() error {
		res.Result = &pb.DataResult_End{&pb.StreamEnd{}}
		return w.WriteMsg(&res)
	}

	writeData := func(key string, data []byte) error {
		res.Result = &pb.DataResult_Data{&pb.DataObject{key, data}}
		return w.WriteMsg(&res)
	}

	for {
		err := r.ReadMsg(&req)
		if err != nil {
			return
		}

		log.Printf("node/data: %s asked for %d objects", pid.Pretty(), len(req.Keys))

		for _, key58 := range req.Keys {
			key, err := multihash.FromB58String(key58)
			if err != nil {
				writeError(err)
				return
			}

			data, err := node.ds.Get(Key(key))
			if err != nil {
				writeError(err)
				return
			}

			if data != nil {
				err = writeData(key58, data)
				if err != nil {
					return
				}
			}
		}

		err = writeEnd()
		if err != nil {
			return
		}

		req.Reset()
	}
}

func (node *Node) registerPeer(ctx context.Context) {
	for {
		err := node.registerPeerImpl(ctx)
		if err == nil {
			return
		}

		// sleep and retry
		select {
		case <-ctx.Done():
			return

		case <-time.After(5 * time.Minute):
			log.Println("Retrying to register with directory")
			continue
		}
	}
}

func (node *Node) registerPeerImpl(ctx context.Context) error {
	err := node.host.Connect(node.netCtx, *node.dir)
	if err != nil {
		log.Printf("Failed to connect to directory: %s", err.Error())
		return err
	}

	s, err := node.host.NewStream(ctx, node.dir.ID, "/mediachain/dir/register")
	if err != nil {
		log.Printf("Failed to open directory stream: %s", err.Error())
		return err
	}
	defer s.Close()

	var pinfo = p2p_pstore.PeerInfo{ID: node.ID}
	var pbpi pb.PeerInfo

	w := ggio.NewDelimitedWriter(s)
	for {
		addrs := node.publicAddrs()

		if len(addrs) > 0 {
			pinfo.Addrs = addrs
			mc.PBFromPeerInfo(&pbpi, pinfo)
			msg := pb.RegisterPeer{&pbpi}

			err = w.WriteMsg(&msg)
			if err != nil {
				log.Printf("Failed to register with directory: %s", err.Error())
				return err
			}
		} else {
			log.Printf("Skipped directory registration; no public address")
		}

		select {
		case <-ctx.Done():
			return nil

		case <-time.After(5 * time.Minute):
			continue
		}
	}
}

// The notion of public address is relative to the network location of the directory
// We want to support directories running on localhost (testing) or in a private network,
// and on the same time not leak internal addresses in public directory announcements.
// So, depending on the directory ip address:
//  If the directory is on the localhost, return the localhost address reported
//   by the host
//  If the directory is in a private range, filter Addrs reported by the
//   host, dropping unroutable addresses
//  If the directory is in a public range, then
//   If the NAT config is manual, return the configured address
//   If the NAT is auto or none, filter the addresses returned by the host,
//      and return only public addresses
func (node *Node) publicAddrs() []multiaddr.Multiaddr {
	if node.status == StatusOffline || node.dir == nil {
		return nil
	}

	dir := node.dir.Addrs[0]
	switch {
	case mc.IsLocalhostAddr(dir):
		return mc.FilterAddrs(node.host.Addrs(), mc.IsLocalhostAddr)

	case mc.IsPrivateAddr(dir):
		return mc.FilterAddrs(node.host.Addrs(), mc.IsRoutableAddr)

	default:
		switch node.natCfg.Opt {
		case mc.NATConfigManual:
			addr := node.natAddr()
			if addr == nil {
				return nil
			}
			return []multiaddr.Multiaddr{addr}

		default:
			return mc.FilterAddrs(node.host.Addrs(), mc.IsPublicAddr)
		}
	}
}

// netAddrs retrieves all routable addresses for the node, regardless of directory
// this includes the auto-detected or manually configured NAT address
func (node *Node) netAddrs() []multiaddr.Multiaddr {
	if node.status == StatusOffline {
		return nil
	}

	addrs := mc.FilterAddrs(node.host.Addrs(), mc.IsRoutableAddr)

	nataddr := node.natAddr()
	if nataddr != nil {
		addrs = append(addrs, nataddr)
	}

	return addrs
}

func (node *Node) natAddr() multiaddr.Multiaddr {
	if node.natCfg.Opt != mc.NATConfigManual {
		return nil
	}

	addr, err := node.natCfg.PublicAddr(node.laddr)
	if err != nil {
		log.Printf("Error determining pubic address: %s", err.Error())
	}
	return addr
}

func (node *Node) doRemoteId(ctx context.Context, pid p2p_peer.ID) (empty NodeInfo, err error) {
	err = node.doConnect(ctx, pid)
	if err != nil {
		return empty, err
	}

	s, err := node.host.NewStream(ctx, pid, "/mediachain/node/id")
	if err != nil {
		return empty, err
	}
	defer s.Close()

	w := ggio.NewDelimitedWriter(s)
	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)

	var req pb.NodeInfoRequest
	var res pb.NodeInfo

	err = w.WriteMsg(&req)
	if err != nil {
		return empty, err
	}

	err = r.ReadMsg(&res)
	if err != nil {
		return empty, err
	}

	return NodeInfo{res.Peer, res.Publisher, res.Info}, nil
}

func (node *Node) doPing(ctx context.Context, pid p2p_peer.ID) error {
	err := node.doConnect(ctx, pid)
	if err != nil {
		return err
	}

	s, err := node.host.NewStream(ctx, pid, "/mediachain/node/ping")
	if err != nil {
		return err
	}
	defer s.Close()

	var ping pb.Ping
	var pong pb.Pong

	w := ggio.NewDelimitedWriter(s)
	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)

	err = w.WriteMsg(&ping)
	if err != nil {
		return err
	}

	return r.ReadMsg(&pong)
}

func (node *Node) doConnect(ctx context.Context, pid p2p_peer.ID) error {
	if node.status == StatusOffline {
		return NodeOffline
	}

	addrs := node.host.Peerstore().Addrs(pid)
	if len(addrs) > 0 {
		return node.host.Connect(node.netCtx, p2p_pstore.PeerInfo{pid, nil})
	}

	pinfo, err := node.doLookup(ctx, pid)
	if err != nil {
		return err
	}

	return node.host.Connect(node.netCtx, pinfo)
}

func (node *Node) doLookup(ctx context.Context, pid p2p_peer.ID) (empty p2p_pstore.PeerInfo, err error) {
	if node.status == StatusOffline {
		return empty, NodeOffline
	}

	if node.dir == nil {
		return empty, NoDirectory
	}

	node.host.Connect(node.netCtx, *node.dir)
	if err != nil {
		return empty, err
	}

	s, err := node.host.NewStream(ctx, node.dir.ID, "/mediachain/dir/lookup")
	if err != nil {
		return empty, err
	}
	defer s.Close()

	req := pb.LookupPeerRequest{pid.Pretty()}
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

func (node *Node) doDirList(ctx context.Context) ([]string, error) {
	if node.status == StatusOffline {
		return nil, NodeOffline
	}

	if node.dir == nil {
		return nil, NoDirectory
	}

	err := node.host.Connect(node.netCtx, *node.dir)
	if err != nil {
		return nil, err
	}

	s, err := node.host.NewStream(ctx, node.dir.ID, "/mediachain/dir/list")
	if err != nil {
		return nil, err
	}
	defer s.Close()

	w := ggio.NewDelimitedWriter(s)
	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)

	var req pb.ListPeersRequest
	var res pb.ListPeersResponse

	err = w.WriteMsg(&req)
	if err != nil {
		return nil, err
	}

	err = r.ReadMsg(&res)
	if err != nil {
		return nil, err
	}

	return res.Peers, nil
}

func (node *Node) doRemoteQuery(ctx context.Context, pid p2p_peer.ID, q string) (<-chan interface{}, error) {
	err := node.doConnect(ctx, pid)
	if err != nil {
		return nil, err
	}

	s, err := node.host.NewStream(ctx, pid, "/mediachain/node/query")
	if err != nil {
		return nil, err
	}

	req := pb.QueryRequest{q}
	w := ggio.NewDelimitedWriter(s)
	err = w.WriteMsg(&req)
	if err != nil {
		s.Close()
		return nil, err
	}

	ch := make(chan interface{})
	go node.doRemoteQueryStream(ctx, s, ch)

	return ch, nil
}

func (node *Node) doRemoteQueryStream(ctx context.Context, s p2p_net.Stream, ch chan interface{}) {
	defer s.Close()
	defer close(ch)

	var res pb.QueryResult
	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)

	for {
		err := r.ReadMsg(&res)
		if err != nil {
			sendStreamError(ctx, ch, err.Error())
			return
		}

		switch res := res.Result.(type) {
		case *pb.QueryResult_Value:
			rv, err := mc.ValueOf(res.Value)
			if err != nil {
				sendStreamError(ctx, ch, err.Error())
				return
			}

			select {
			case ch <- rv:
			case <-ctx.Done():
				return
			}

		case *pb.QueryResult_End:
			return

		case *pb.QueryResult_Error:
			sendStreamError(ctx, ch, res.Error.Error)
			return

		default:
			sendStreamError(ctx, ch, "Unexpected result")
			return
		}

		res.Reset()
	}
}

func (node *Node) doMerge(ctx context.Context, pid p2p_peer.ID, q string) (count int, ocount int, err error) {
	ch, err := node.doRemoteQuery(ctx, pid, q)
	if err != nil {
		return 0, 0, err
	}

	// publisher key cache
	pkcache := make(map[string]p2p_crypto.PubKey)

	// background data merges
	workers := runtime.NumCPU()
	workch := make(chan map[string]Key, 64*workers) // ~ 3MB/worker
	resch := make(chan MergeResult, workers)
	for x := 0; x < workers; x++ {
		go node.doMergeDataAsync(ctx, pid, workch, resch)
	}

	const batch = 1024
	stmts := make([]*pb.Statement, 0, batch)
	keys := make(map[string]Key)

loop:
	for val := range ch {
		switch val := val.(type) {
		case *pb.Statement:
			if !node.checkStatement(val) {
				err = BadStatement
				break loop
			}

			var verify bool
			verify, err = node.verifyStatementCacheKeys(val, pkcache)
			if err != nil {
				break loop
			}

			// a verification failure taints the result set; abort the merge
			if !verify {
				err = BadStatement
				break loop
			}

			err = node.mergeStatementKeys(val, keys)
			if err != nil {
				break loop
			}

			if len(keys) >= batch {
				select {
				case workch <- keys:
					keys = make(map[string]Key)

				case res := <-resch:
					ocount += res.count
					err = res.err
					workers -= 1
					break loop

				case <-ctx.Done():
					err = ctx.Err()
					break loop
				}
			}

			stmts = append(stmts, val)

			if len(stmts) >= batch {
				var xcount int
				xcount, err = node.db.MergeBatch(stmts)
				count += xcount
				if err != nil {
					break loop
				}
				stmts = stmts[:0]
			}

		case StreamError:
			err = val
			break loop

		default:
			err = BadResult
			break loop
		}
	}

	if len(keys) > 0 && err == nil {
		select {
		case workch <- keys:

		case res := <-resch:
			ocount += res.count
			err = res.err
			workers -= 1

		case <-ctx.Done():
			err = ctx.Err()
		}
	}

	if len(stmts) > 0 && err == nil {
		var xcount int
		xcount, err = node.db.MergeBatch(stmts)
		count += xcount
	}

	close(workch)
	for x := 0; x < workers; x++ {
		res := <-resch
		ocount += res.count
		if err == nil && res.err != nil {
			err = res.err
		}
	}

	return count, ocount, err
}

type MergeResult struct {
	count int
	err   error
}

func (node *Node) doMergeDataAsync(ctx context.Context, pid p2p_peer.ID,
	in <-chan map[string]Key,
	out chan<- MergeResult) {
	var s p2p_net.Stream
	var err error
	var count int

	for keys := range in {
		if s == nil {
			s, err = node.host.NewStream(ctx, pid, "/mediachain/node/data")
			if err != nil {
				break
			}
			defer s.Close()
		}

		var xcount int
		xcount, err = node.doMergeDataImpl(s, keys)
		count += xcount
		if err != nil {
			break
		}
	}

	out <- MergeResult{count, err}
}

func (node *Node) doMergeDataImpl(s p2p_net.Stream, keys map[string]Key) (count int, err error) {
	keys58 := make([]string, 0, len(keys))
	for key58, _ := range keys {
		keys58 = append(keys58, key58)
	}

	var req pb.DataRequest
	var res pb.DataResult

	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	w := ggio.NewDelimitedWriter(s)

	req.Keys = keys58
	err = w.WriteMsg(&req)
	if err != nil {
		return 0, err
	}

loop:
	for {
		err := r.ReadMsg(&res)

		switch res := res.Result.(type) {
		case *pb.DataResult_Data:
			key58 := res.Data.Key

			key, ok := keys[key58]
			if !ok {
				return count, UnexpectedData
			}

			// verify data hash
			data := res.Data.Data
			hash := mc.Hash(data)
			if !bytes.Equal([]byte(key), []byte(hash)) {
				return count, BadData
			}

			_, err = node.ds.Put(data)
			if err != nil {
				return count, err
			}

			delete(keys, key58)
			count++

		case *pb.DataResult_End:
			break loop

		case *pb.DataResult_Error:
			return count, StreamError{res.Error.Error}

		default:
			return count, BadResult
		}

		res.Reset()
	}

	if len(keys) > 0 { // we didn't get all the data we asked for, signal error
		return count, MissingData
	}

	return count, nil
}

func (node *Node) mergeStatementKeys(stmt *pb.Statement, keys map[string]Key) error {
	mergeSimple := func(s *pb.SimpleStatement) error {
		err := node.mergeObjectKey(s.Object, keys)
		if err != nil {
			return err
		}

		for _, dep := range s.Deps {
			err = node.mergeObjectKey(dep, keys)
			if err != nil {
				return err
			}
		}

		return nil
	}

	switch body := stmt.Body.Body.(type) {
	case *pb.StatementBody_Simple:
		return mergeSimple(body.Simple)

	case *pb.StatementBody_Compound:
		ss := body.Compound.Body
		for _, s := range ss {
			err := mergeSimple(s)
			if err != nil {
				return err
			}
		}
		return nil

	case *pb.StatementBody_Envelope:
		stmts := body.Envelope.Body
		for _, stmt := range stmts {
			err := node.mergeStatementKeys(stmt, keys)
			if err != nil {
				return err
			}
		}
		return nil

	default:
		return BadStatementBody
	}
}

func (node *Node) mergeObjectKey(key58 string, keys map[string]Key) error {
	_, have := keys[key58]
	if have {
		return nil
	}

	mhash, err := multihash.FromB58String(key58)
	if err != nil {
		return err
	}

	key := Key(mhash)
	have, err = node.ds.Has(key)
	if err != nil {
		return err
	}

	if !have {
		keys[key58] = key
	}

	return nil
}

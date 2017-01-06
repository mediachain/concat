package main

import (
	"bytes"
	"context"
	ggio "github.com/gogo/protobuf/io"
	p2p_crypto "github.com/libp2p/go-libp2p-crypto"
	p2p_net "github.com/libp2p/go-libp2p-net"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	mc "github.com/mediachain/concat/mc"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
	multihash "github.com/multiformats/go-multihash"
	"log"
	"runtime"
)

func (node *Node) pingHandler(s p2p_net.Stream) {
	// DEPRECATED in favor of libp2p.PingService
	defer s.Close()

	pid := mc.LogStreamHandler(s)

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

	mc.LogStreamHandler(s)

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

	pid := mc.LogStreamHandler(s)

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
				log.Printf("node/query: error constructing value: %s", err.Error())
				writeError(err)
				return err
			}

			res.Result = &pb.QueryResult_Value{&pb.QueryResultValue{
				&pb.QueryResultValue_Compound{cv}}}

		default:
			sv, err := mc.SimpleValue(val)
			if err != nil {
				log.Printf("node/query: error constructing value: %s", err.Error())
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

	pid := mc.LogStreamHandler(s)

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

func (node *Node) pushHandler(s p2p_net.Stream) {
	defer s.Close()

	pid := mc.LogStreamHandler(s)

	var err error
	var req pb.PushRequest
	var res pb.PushResponse
	var val pb.PushValue
	var end pb.PushEnd

	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	w := ggio.NewDelimitedWriter(s)

	err = r.ReadMsg(&req)
	if err != nil {
		return
	}

	if !node.auth.authorize(pid, req.Namespaces) {
		log.Printf("node/push: rejected push from %s; not authorized", pid.Pretty())
		res.Body = &pb.PushResponse_Reject{&pb.PushReject{"Not authorized"}}
		w.WriteMsg(&res)
		return
	}

	res.Body = &pb.PushResponse_Accept{&pb.PushAccept{}}
	err = w.WriteMsg(&res)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(node.netCtx)
	defer cancel()

	wch := make(chan interface{})
	rch := make(chan PushMergeResult, 1)
	var mres PushMergeResult
	var mdone bool

	go func() {
		scount, ocount, err := node.doMergeStream(ctx, pid, wch)
		rch <- PushMergeResult{scount, ocount, err}
	}()

	nsfilter := make(map[string]bool)
	for _, ns := range req.Namespaces {
		nsfilter[ns] = true
	}

loop:
	for {
		err = r.ReadMsg(&val)
		if err != nil {
			break loop
		}

		switch val := val.Value.(type) {
		case *pb.PushValue_Stmt:
			if val.Stmt == nil {
				err = BadPush
				break loop
			}

			if !nsfilter[val.Stmt.Namespace] {
				err = BadPush
				break loop
			}

			select {
			case wch <- val.Stmt:
			case mres = <-rch:
				mdone = true
				break loop
			}

		case *pb.PushValue_End:
			break loop

		default:
			err = BadPush
			break loop
		}

		val.Reset()
	}

	close(wch)
	if !mdone {
		mres = <-rch
	}

	log.Printf("node/push: merged %d statements and %d objects from %s", mres.scount, mres.ocount, pid.Pretty())

	end.Statements = int64(mres.scount)
	end.Objects = int64(mres.ocount)
	if err == nil && mres.err != nil {
		err = mres.err
	}
	if err != nil {
		end.Error = err.Error()
		log.Printf("node/push: merge error: %s", end.Error)
	}

	w.WriteMsg(&end)
}

type PushMergeResult struct {
	scount int
	ocount int
	err    error
}

func (node *Node) doRemoteId(ctx context.Context, pid p2p_peer.ID) (empty NodeInfo, err error) {
	s, err := node.doConnect(ctx, pid, "/mediachain/node/id")
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
	// DEPRECATED in favor of libp2p.PingService; see netPing
	s, err := node.doConnect(ctx, pid, "/mediachain/node/ping")
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

func (node *Node) doRemoteQuery(ctx context.Context, pid p2p_peer.ID, q string) (<-chan interface{}, error) {
	s, err := node.doConnect(ctx, pid, "/mediachain/node/query")
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

	return node.doMergeStream(ctx, pid, ch)
}

func (node *Node) doMergeStream(ctx context.Context, pid p2p_peer.ID, ch <-chan interface{}) (count int, ocount int, err error) {
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

// Note: it is possible to refetch the same object if it appears in multiple batches.
// This is complicated to dedupe, as it would require keeping a synchronous map
// tracking in flight fetches (and consulting it when merging object keys)
// On the other hand, in the standard usage this should only happen for
// schema objects, and would result in at most NumCPU dupe fetches.
// So the overhead should be minimal and not worth the complexity/slowdown from
// tracking in-flight requests
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
	for key58, key := range keys {
		have, err := node.ds.Has(key)
		if err != nil {
			return 0, err
		}
		if have {
			continue
		}
		keys58 = append(keys58, key58)
	}

	if len(keys58) == 0 {
		return 0, nil
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
		if err != nil {
			return count, err
		}

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

	if count < len(keys58) { // we didn't get all the data we asked for, signal error
		return count, MissingData
	}

	return count, nil
}

func (node *Node) doRawMerge(ctx context.Context, pid p2p_peer.ID, keys map[string]Key) (int, error) {
	s, err := node.doConnect(ctx, pid, "/mediachain/node/data")
	if err != nil {
		return 0, err
	}
	defer s.Close()

	return node.doMergeDataImpl(s, keys)
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

	keys[key58] = Key(mhash)
	return nil
}

func (node *Node) doPush(ctx context.Context, pid p2p_peer.ID, q *mcq.Query) (int, int, error) {
	nsq := q.WithSimpleSelect("namespace")
	nsr, err := node.db.Query(nsq)
	if err != nil {
		return 0, 0, err
	}

	if len(nsr) == 0 {
		return 0, 0, err
	}

	nss := make([]string, len(nsr))
	for x, ns := range nsr {
		nss[x] = ns.(string)
	}

	s, err := node.doConnect(ctx, pid, "/mediachain/node/push")
	if err != nil {
		return 0, 0, err
	}
	defer s.Close()

	var req pb.PushRequest
	var res pb.PushResponse

	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	w := ggio.NewDelimitedWriter(s)

	req.Namespaces = nss

	err = w.WriteMsg(&req)
	if err != nil {
		return 0, 0, err
	}

	err = r.ReadMsg(&res)
	if err != nil {
		return 0, 0, err
	}

	switch body := res.Body.(type) {
	case *pb.PushResponse_Accept:
		break
	case *pb.PushResponse_Reject:
		return 0, 0, PushError(body.Reject.Error)
	default:
		return 0, 0, BadResponse
	}

	qch, err := node.db.QueryStream(ctx, q)
	if err != nil {
		return 0, 0, err
	}

	rch := make(chan PushMergeResult, 1)

	go func() {
		var end pb.PushEnd
		err := r.ReadMsg(&end)
		if err != nil {
			rch <- PushMergeResult{-1, -1, err}
			return
		}

		var res PushMergeResult
		res.scount = int(end.Statements)
		res.ocount = int(end.Objects)
		if end.Error != "" {
			res.err = PushError(end.Error)
		}
		rch <- res
	}()

	var val pb.PushValue

	writeValue := func(stmt *pb.Statement) error {
		val.Value = &pb.PushValue_Stmt{stmt}
		return w.WriteMsg(&val)
	}

	writeEnd := func() error {
		val.Value = &pb.PushValue_End{&pb.StreamEnd{}}
		return w.WriteMsg(&val)
	}

loop:
	for {
		select {
		case stmt, ok := <-qch:
			if !ok {
				break loop
			}

			err = writeValue(stmt.(*pb.Statement))
			if err != nil {
				return -1, -1, err
			}

		case res := <-rch:
			log.Printf("node/push: premature push end: %s", res.err)
			return res.scount, res.ocount, res.err

		case <-ctx.Done():
			break loop
		}
	}

	err = writeEnd()
	if err != nil {
		return -1, -1, err
	}

	pres := <-rch
	return pres.scount, pres.ocount, pres.err
}

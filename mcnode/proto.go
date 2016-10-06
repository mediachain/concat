package main

import (
	"context"
	"errors"
	ggio "github.com/gogo/protobuf/io"
	p2p_net "github.com/libp2p/go-libp2p-net"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	p2p_pstore "github.com/libp2p/go-libp2p-peerstore"
	mc "github.com/mediachain/concat/mc"
	mcq "github.com/mediachain/concat/mc/query"
	pb "github.com/mediachain/concat/proto"
	multiaddr "github.com/multiformats/go-multiaddr"
	"log"
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
		node.status = StatusOffline
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
	host, err := mc.NewHost(node.NodeIdentity, node.laddr)
	if err != nil {
		return err
	}

	host.SetStreamHandler("/mediachain/node/ping", node.pingHandler)
	host.SetStreamHandler("/mediachain/node/query", node.queryHandler)
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
		go node.registerPeer(node.netCtx, node.laddr)
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
	log.Printf("node/ping: new stream from %s", pid.Pretty())

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

func (node *Node) queryHandler(s p2p_net.Stream) {
	defer s.Close()
	pid := s.Conn().RemotePeer()
	log.Printf("node/query: new stream from %s", pid.Pretty())

	ctx, cancel := context.WithCancel(node.netCtx)
	defer cancel()

	var req pb.QueryRequest
	var res pb.QueryResult

	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	w := ggio.NewDelimitedWriter(s)

	writeError := func(err error) {
		res.Result = &pb.QueryResult_Error{&pb.QueryResultError{err.Error()}}
		w.WriteMsg(&res)
	}

	writeEnd := func() error {
		res.Result = &pb.QueryResult_End{&pb.QueryResultEnd{}}
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

func (node *Node) registerPeer(ctx context.Context, addrs ...multiaddr.Multiaddr) {
	for {
		err := node.registerPeerImpl(ctx, addrs...)
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

func (node *Node) registerPeerImpl(ctx context.Context, addrs ...multiaddr.Multiaddr) error {
	err := node.host.Connect(ctx, *node.dir)
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

	pinfo := p2p_pstore.PeerInfo{node.ID, addrs}
	var pbpi pb.PeerInfo
	mc.PBFromPeerInfo(&pbpi, pinfo)
	msg := pb.RegisterPeer{&pbpi}

	w := ggio.NewDelimitedWriter(s)
	for {
		log.Printf("Registering with directory")
		err = w.WriteMsg(&msg)
		if err != nil {
			log.Printf("Failed to register with directory: %s", err.Error())
			return err
		}

		select {
		case <-ctx.Done():
			return nil

		case <-time.After(5 * time.Minute):
			continue
		}
	}
}

func (node *Node) doPing(ctx context.Context, pid p2p_peer.ID) error {
	if node.status == StatusOffline {
		return NodeOffline
	}

	pinfo, err := node.doLookup(ctx, pid)
	if err != nil {
		return err
	}

	err = node.host.Connect(ctx, pinfo)
	if err != nil {
		return err
	}

	s, err := node.host.NewStream(ctx, pinfo.ID, "/mediachain/node/ping")
	if err != nil {
		return err
	}
	defer s.Close()

	var ping pb.Ping
	w := ggio.NewDelimitedWriter(s)
	err = w.WriteMsg(&ping)
	if err != nil {
		return err
	}

	var pong pb.Pong
	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	err = r.ReadMsg(&pong)

	return err
}

func (node *Node) doLookup(ctx context.Context, pid p2p_peer.ID) (empty p2p_pstore.PeerInfo, err error) {
	if node.status == StatusOffline {
		return empty, NodeOffline
	}

	if node.dir == nil {
		return empty, NoDirectory
	}

	node.host.Connect(ctx, *node.dir)
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

func (node *Node) doRemoteQuery(ctx context.Context, pid p2p_peer.ID, q string) (<-chan interface{}, error) {

	if node.status == StatusOffline {
		return nil, NodeOffline
	}

	pinfo, err := node.doLookup(ctx, pid)
	if err != nil {
		return nil, err
	}

	err = node.host.Connect(ctx, pinfo)
	if err != nil {
		return nil, err
	}

	s, err := node.host.NewStream(ctx, pinfo.ID, "/mediachain/node/query")
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

func (node *Node) doMerge(ctx context.Context, pid p2p_peer.ID, q string) (count int, err error) {
	ch, err := node.doRemoteQuery(ctx, pid, q)
	if err != nil {
		return 0, err
	}

	for val := range ch {
		switch val := val.(type) {
		case *pb.Statement:
			// TODO statement signature verification
			ins, err := node.db.Merge(val)
			if err != nil {
				return count, err
			}
			if ins {
				count += 1
			}

		case StreamError:
			return count, val

		default:
			return count, BadResult
		}
	}

	return count, nil
}

package main

import (
	"context"
	"errors"
	ggio "github.com/gogo/protobuf/io"
	p2p_peer "github.com/ipfs/go-libp2p-peer"
	p2p_pstore "github.com/ipfs/go-libp2p-peerstore"
	multiaddr "github.com/jbenet/go-multiaddr"
	p2p_net "github.com/libp2p/go-libp2p/p2p/net"
	mc "github.com/mediachain/concat/mc"
	pb "github.com/mediachain/concat/proto"
	"log"
	"time"
)

var NodeOffline = errors.New("Node is offline")
var NoDirectory = errors.New("No directory server")
var UnknownPeer = errors.New("Unknown peer")

func (node *Node) goOffline() error {
	return nil
}

func (node *Node) goOnline() error {
	node.mx.Lock()
	defer node.mx.Unlock()

	switch node.status {
	case StatusOffline:
		host, err := mc.NewHost(node.Identity, node.laddr)
		if err != nil {
			return err
		}

		host.SetStreamHandler("/mediachain/node/ping", node.pingHandler)
		node.host = host
		node.status = StatusOnline
		log.Printf("Node is online\n")
		return nil
	}

	return nil
}

func (node *Node) goPublic() error {
	return nil
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

func (node *Node) registerPeer(addrs ...multiaddr.Multiaddr) {
	// directory failure is a fatality for now
	ctx := context.Background()

	err := node.host.Connect(ctx, *node.dir)
	if err != nil {
		log.Printf("Failed to connect to directory")
		log.Fatal(err)
	}

	s, err := node.host.NewStream(ctx, node.dir.ID, "/mediachain/dir/register")
	if err != nil {
		log.Printf("Failed to open directory stream")
		log.Fatal(err)
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
			log.Printf("Failed to register with directory")
			log.Fatal(err)
		}

		time.Sleep(5 * time.Minute)
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

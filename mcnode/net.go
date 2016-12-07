package main

import (
	"context"
	ggio "github.com/gogo/protobuf/io"
	p2p_net "github.com/libp2p/go-libp2p-net"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	p2p_pstore "github.com/libp2p/go-libp2p-peerstore"
	mc "github.com/mediachain/concat/mc"
	pb "github.com/mediachain/concat/proto"
	multiaddr "github.com/multiformats/go-multiaddr"
	"log"
	"time"
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

		err := node.dht.Close()
		if err != nil {
			log.Printf("Error closing DHT: %s", err.Error())
		}
		node.dht = nil

		err = node.host.Close()
		if err != nil {
			log.Printf("Error closing host: %s", err.Error())
		}
		node.host = nil

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

	ctx, cancel := context.WithCancel(context.Background())
	host, err := mc.NewHost(ctx, node.PeerIdentity, node.laddr, opts...)
	if err != nil {
		cancel()
		return err
	}

	host.SetStreamHandler("/mediachain/node/id", node.idHandler)
	host.SetStreamHandler("/mediachain/node/ping", node.pingHandler)
	host.SetStreamHandler("/mediachain/node/query", node.queryHandler)
	host.SetStreamHandler("/mediachain/node/data", node.dataHandler)
	host.SetStreamHandler("/mediachain/node/push", node.pushHandler)

	dht := NewDHT(ctx, host)

	err = dht.Bootstrap()
	if err != nil {
		// that's non-fatal, it will just fail to lookup
		log.Printf("Error boostrapping DHT: %s", err.Error())
	}

	node.host = host
	node.netCtx = ctx
	node.netCancel = cancel
	node.dht = dht

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

		if node.natCfg.Opt == mc.NATConfigAuto {
			// wait a bit for NAT port mapping to take effect
			time.Sleep(2 * time.Second)
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

func (node *Node) netConns() []p2p_pstore.PeerInfo {
	if node.status == StatusOffline {
		return nil
	}

	conns := node.host.Network().Conns()
	peers := make([]p2p_pstore.PeerInfo, len(conns))
	for x, conn := range conns {
		peers[x].ID = conn.RemotePeer()
		peers[x].Addrs = []multiaddr.Multiaddr{conn.RemoteMultiaddr()}
	}

	return peers
}

// Connectivity
func (node *Node) doConnect(ctx context.Context, pid p2p_peer.ID) error {
	if node.status == StatusOffline {
		return NodeOffline
	}

	if node.host.Network().Connectedness(pid) == p2p_net.Connected {
		return nil
	}

	addrs := node.host.Peerstore().Addrs(pid)
	if len(addrs) > 0 {
		return node.host.Connect(ctx, p2p_pstore.PeerInfo{pid, addrs})
	}

	pinfo, err := node.doLookup(ctx, pid)
	if err != nil {
		return err
	}

	return node.host.Connect(node.netCtx, pinfo)
}

func (node *Node) doLookup(ctx context.Context, pid p2p_peer.ID) (pinfo p2p_pstore.PeerInfo, err error) {
	pinfo, err = node.doLookupImpl(ctx, pid)
	if err == nil {
		node.host.Peerstore().AddAddrs(pid, pinfo.Addrs, p2p_pstore.AddressTTL)
	}

	return
}

func (node *Node) doLookupImpl(ctx context.Context, pid p2p_peer.ID) (pinfo p2p_pstore.PeerInfo, err error) {
	if node.status == StatusOffline {
		return pinfo, NodeOffline
	}

	if node.dir == nil {
		goto lookup_dht
	}

	pinfo, err = node.doDirLookup(ctx, pid)
	if err == nil {
		return
	}

	if err != UnknownPeer {
		log.Printf("Directory lookup error: %s", err.Error())
	}

lookup_dht:
	return node.dht.Lookup(ctx, pid)
}

func (node *Node) doDirLookup(ctx context.Context, pid p2p_peer.ID) (empty p2p_pstore.PeerInfo, err error) {
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

package main

import (
	"context"
	ggio "github.com/gogo/protobuf/io"
	p2p_net "github.com/libp2p/go-libp2p-net"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	p2p_pstore "github.com/libp2p/go-libp2p-peerstore"
	p2p_proto "github.com/libp2p/go-libp2p-protocol"
	p2p_ping "github.com/libp2p/go-libp2p/p2p/protocol/ping"
	mc "github.com/mediachain/concat/mc"
	mcq "github.com/mediachain/concat/mc/query"
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
		node.ping = nil

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
	host.SetStreamHandler("/mediachain/node/manifest", node.manifestHandler)
	host.SetStreamHandler("/mediachain/node/ping", node.pingHandler)
	host.SetStreamHandler("/mediachain/node/query", node.queryHandler)
	host.SetStreamHandler("/mediachain/node/data", node.dataHandler)
	host.SetStreamHandler("/mediachain/node/push", node.pushHandler)

	ping := p2p_ping.NewPingService(host)

	dht := NewDHT(ctx, host)

	err = dht.Bootstrap()
	if err != nil {
		// that's non-fatal, it will just fail to lookup
		log.Printf("Error boostrapping DHT: %s", err.Error())
	}

	node.host = host
	node.netCtx = ctx
	node.netCancel = cancel
	node.ping = ping
	node.dht = dht

	return nil
}

// goPublic starts the network if it's not already up and registers with the
// directory; fails with NoDirectory if that hasn't been configured.
func (node *Node) goPublic() error {
	if len(node.dir) == 0 {
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
		for _, dir := range node.dir {
			go node.registerPeer(node.netCtx, dir)
		}
		node.status = StatusPublic

		log.Println("Node is public")
		return nil

	default:
		return nil
	}
}

func (node *Node) registerPeer(ctx context.Context, dir p2p_pstore.PeerInfo) {
	for {
		err := node.registerPeerImpl(ctx, dir)
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

func (node *Node) registerPeerImpl(ctx context.Context, dir p2p_pstore.PeerInfo) error {
	s, err := node.doDirConnect(node.netCtx, dir, "/mediachain/dir/register")
	if err != nil {
		log.Printf("Failed to connect to directory: %s", err.Error())
		return err
	}
	defer s.Close()

	var pinfo = p2p_pstore.PeerInfo{ID: node.ID}
	var pbpi pb.PeerInfo
	var pbpub = pb.PublisherInfo{Id: node.publisher.ID58}

	w := ggio.NewDelimitedWriter(s)
	for {
		addrs := node.publicAddrs(dir)

		if len(addrs) > 0 {
			pinfo.Addrs = addrs
			mc.PBFromPeerInfo(&pbpi, pinfo)

			ns := node.publicNamespaces()
			pbpub.Namespaces = ns

			mfs := node.mfs

			msg := pb.RegisterPeer{&pbpi, &pbpub, mfs}

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

func (node *Node) publicNamespaces() []string {
	res, err := node.db.Query(nsQuery)
	if err != nil {
		log.Printf("Namespace query error: %s", err.Error())
		return nil
	}

	pns := make([]string, len(res))
	for x, ns := range res {
		pns[x] = ns.(string)
	}

	return pns
}

var nsQuery *mcq.Query

func init() {
	q, err := mcq.ParseQuery("SELECT namespace FROM *")
	if err != nil {
		log.Fatal(err)
	}
	nsQuery = q
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
func (node *Node) publicAddrs(dir p2p_pstore.PeerInfo) []multiaddr.Multiaddr {
	if node.status == StatusOffline {
		return nil
	}

	diraddr := dir.Addrs[0]
	switch {
	case mc.IsLocalhostAddr(diraddr):
		return mc.FilterAddrs(node.host.Addrs(), mc.IsLocalhostAddr)

	case mc.IsPrivateAddr(diraddr):
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

func (node *Node) netPeerAddrs(pid p2p_peer.ID) []multiaddr.Multiaddr {
	if node.status == StatusOffline {
		return nil
	}

	pinfo := node.host.Peerstore().PeerInfo(pid)
	return pinfo.Addrs
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

func (node *Node) netPing(ctx context.Context, pid p2p_peer.ID) (dt time.Duration, err error) {
	err = node.doConnectPeer(ctx, pid)
	if err != nil {
		return
	}

	ch, err := node.ping.Ping(ctx, pid)
	if err != nil {
		return
	}

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return

	case dt = <-ch:
		return
	}
}

func (node *Node) netIdentify(ctx context.Context, pid p2p_peer.ID) (nid NetIdentify, err error) {
	if node.status == StatusOffline {
		return nid, NodeOffline
	}

	if node.host.Network().Connectedness(pid) == p2p_net.Connected {
		goto identify
	}

	err = node.doConnectPeer(ctx, pid)
	if err != nil {
		return
	}

	// give libp2p some time to populate the peerstore
	time.Sleep(1 * time.Second)

identify:
	pstore := node.host.Peerstore()
	nid.ID = pid.Pretty()

	pubk := pstore.PubKey(pid)
	if pubk != nil {
		bytes, err := pubk.Bytes()
		if err != nil {
			return nid, err
		}
		nid.PublicKey = bytes
	}

	maddrs := pstore.Addrs(pid)
	saddrs := make([]string, len(maddrs))
	for x, maddr := range maddrs {
		saddrs[x] = maddr.String()
	}
	nid.Addresses = saddrs

	cver, err := pstore.Get(pid, "AgentVersion")
	cvers, ok := cver.(string)
	if ok {
		nid.AgentVersion = cvers
	}

	pver, err := pstore.Get(pid, "ProtocolVersion")
	pvers, ok := pver.(string)
	if ok {
		nid.ProtocolVersion = pvers
	}

	protos, err := pstore.GetProtocols(pid)
	if err != nil {
		return
	}
	nid.Protocols = protos

	return
}

// Connectivity
func (node *Node) doConnect(ctx context.Context, pid p2p_peer.ID, proto p2p_proto.ID) (p2p_net.Stream, error) {
	err := node.doConnectPeer(ctx, pid)
	if err != nil {
		return nil, err
	}

	return node.host.NewStream(ctx, pid, proto)
}

func (node *Node) doConnectPeer(ctx context.Context, pid p2p_peer.ID) error {
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
		node.host.Peerstore().AddAddrs(pid, pinfo.Addrs, p2p_pstore.ProviderAddrTTL)
	}

	return
}

func (node *Node) doLookupImpl(ctx context.Context, pid p2p_peer.ID) (empty p2p_pstore.PeerInfo, err error) {
	if node.status == StatusOffline {
		return empty, NodeOffline
	}

	dirs := node.dir
	workers := len(dirs) + 1
	ch := make(chan interface{}, workers)

	var dirctx, dhtctx context.Context
	var dircancel, dhtcancel context.CancelFunc

	if len(dirs) == 0 {
		goto lookup_dht
	}

	dirctx, dircancel = context.WithTimeout(ctx, 5*time.Second)
	defer dircancel()

	for _, dir := range dirs {
		go func(dir p2p_pstore.PeerInfo) {
			pinfo, err := node.doDirLookupImpl(dirctx, dir, pid)
			if err == nil {
				ch <- pinfo
			} else {
				ch <- err
			}
		}(dir)
	}

lookup_dht:
	dhtctx, dhtcancel = context.WithTimeout(ctx, 30*time.Second)
	defer dhtcancel()

	go func() {
		pinfo, err := node.dht.Lookup(dhtctx, pid)
		if err == nil {
			ch <- pinfo
		} else {
			ch <- err
		}
	}()

	errcount := 0
loop:
	for x := 0; x < workers; x++ {
		select {
		case res := <-ch:
			switch res := res.(type) {
			case p2p_pstore.PeerInfo:
				return res, nil

			case error:
				if res != UnknownPeer {
					log.Printf("Peer lookup error: %s", res.Error())
					errcount++
				}

			default:
				log.Printf("doLookupImpl: unexpected result type: %T", res)
				errcount++
			}

		case <-ctx.Done():
			break loop
		}
	}

	// we didn't resolve, but what kind of error is it?
	if errcount < workers {
		// not all workers failed, or our context expired; we didn't find the peer
		err = UnknownPeer
	} else {
		// all workers failed, signal error
		err = LookupError
	}

	return
}

func (node *Node) doDirLookupImpl(ctx context.Context, dir p2p_pstore.PeerInfo, pid p2p_peer.ID) (empty p2p_pstore.PeerInfo, err error) {
	s, err := node.doDirConnect(ctx, dir, "/mediachain/dir/lookup")
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

func (node *Node) doDirList(ctx context.Context, ns string) ([]string, error) {
	return node.doDirCollect(ctx,
		func(ctx context.Context, dir p2p_pstore.PeerInfo) ([]string, error) {
			return node.doDirListImpl(ctx, dir, ns)
		})
}

func (node *Node) doDirListNS(ctx context.Context) ([]string, error) {
	return node.doDirCollect(ctx, node.doDirListNSImpl)
}

func (node *Node) doDirCollect(ctx context.Context, proc func(ctx context.Context, dir p2p_pstore.PeerInfo) ([]string, error)) ([]string, error) {
	dirs := node.dir

	switch len(dirs) {
	case 0:
		return nil, NoDirectory
	case 1:
		return proc(ctx, dirs[0])
	}

	ch := make(chan interface{}, len(dirs))
	for _, dir := range dirs {
		go func(dir p2p_pstore.PeerInfo) {
			res, err := proc(ctx, dir)
			if err == nil {
				ch <- res
			} else {
				ch <- err
			}
		}(dir)
	}

	var err error
	errcount := 0
	vals := make(map[string]bool)

loop:
	for x, xlen := 0, len(dirs); x < xlen; x++ {
		select {
		case res := <-ch:
			switch res := res.(type) {
			case nil: // empty list

			case []string:
				for _, val := range res {
					vals[val] = true
				}

			case error:
				log.Printf("Directory error: %s", res.Error())
				errcount++

			default:
				log.Printf("doDirCollect: unexpected result type: %T", res)
				errcount++
			}

		case <-ctx.Done():
			err = ctx.Err()
			break loop
		}
	}

	if len(vals) == 0 {
		// we didn't get any values, but is it an error?
		if errcount < len(dirs) {
			// not all directories failed, error only if ctx was done
			return nil, err
		} else {
			// all directories failed, signal error
			return nil, DirectoryError
		}
	}

	res := make([]string, 0, len(vals))
	for val, _ := range vals {
		res = append(res, val)
	}

	return res, nil
}

func (node *Node) doDirListImpl(ctx context.Context, dir p2p_pstore.PeerInfo, ns string) ([]string, error) {
	s, err := node.doDirConnect(ctx, dir, "/mediachain/dir/list")
	if err != nil {
		return nil, err
	}
	defer s.Close()

	w := ggio.NewDelimitedWriter(s)
	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)

	var req pb.ListPeersRequest
	var res pb.ListPeersResponse

	req.Namespace = ns

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

func (node *Node) doDirListNSImpl(ctx context.Context, dir p2p_pstore.PeerInfo) ([]string, error) {
	s, err := node.doDirConnect(ctx, dir, "/mediachain/dir/listns")
	if err != nil {
		return nil, err
	}
	defer s.Close()

	w := ggio.NewDelimitedWriter(s)
	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)

	var req pb.ListNamespacesRequest
	var res pb.ListNamespacesResponse

	err = w.WriteMsg(&req)
	if err != nil {
		return nil, err
	}

	err = r.ReadMsg(&res)
	if err != nil {
		return nil, err
	}

	return res.Namespaces, nil
}

func (node *Node) doDirConnect(ctx context.Context, dir p2p_pstore.PeerInfo, proto p2p_proto.ID) (p2p_net.Stream, error) {
	if node.status == StatusOffline {
		return nil, NodeOffline
	}

	err := node.host.Connect(node.netCtx, dir)
	if err != nil {
		return nil, err
	}

	return node.host.NewStream(ctx, dir.ID, proto)
}

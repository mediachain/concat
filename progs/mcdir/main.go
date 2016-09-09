package main

import (
	"context"
	ggio "github.com/gogo/protobuf/io"
	p2p_crypto "github.com/ipfs/go-libp2p-crypto"
	p2p_peer "github.com/ipfs/go-libp2p-peer"
	p2p_pstore "github.com/ipfs/go-libp2p-peerstore"
	multiaddr "github.com/jbenet/go-multiaddr"
	p2p_host "github.com/libp2p/go-libp2p/p2p/host"
	p2p_bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	p2p_metrics "github.com/libp2p/go-libp2p/p2p/metrics"
	p2p_net "github.com/libp2p/go-libp2p/p2p/net"
	p2p_swarm "github.com/libp2p/go-libp2p/p2p/net/swarm"
	pb "github.com/mediachain/concat/proto"
	"log"
	"sync"
)

type Directory struct {
	pkey  p2p_crypto.PrivKey
	id    p2p_peer.ID
	host  p2p_host.Host
	peers map[p2p_peer.ID]p2p_pstore.PeerInfo
	mx    sync.Mutex
}

const MaxMessageSize = 2 << 20 // 1 MB

func pbToPeerInfo(pbpi *pb.PeerInfo) (empty p2p_pstore.PeerInfo, err error) {
	pid := p2p_peer.ID(pbpi.Id)
	addrs := make([]multiaddr.Multiaddr, len(pbpi.Addr))
	for x, bytes := range pbpi.Addr {
		addr, err := multiaddr.NewMultiaddrBytes(bytes)
		if err != nil {
			return empty, err
		}
		addrs[x] = addr
	}

	return p2p_pstore.PeerInfo{ID: pid, Addrs: addrs}, nil
}

func (dir *Directory) registerHandler(s p2p_net.Stream) {
	defer s.Close()

	pid := s.Conn().RemotePeer()
	log.Printf("directory/register: new stream from %s\n", pid.Pretty())

	reader := ggio.NewDelimitedReader(s, MaxMessageSize)
	req := new(pb.RegisterPeer)

	for {
		err := reader.ReadMsg(req)
		if err != nil {
			break
		}

		if req.Info == nil {
			log.Printf("directory/register: empty peer info from %s\n", pid.Pretty())
			break
		}

		pinfo, err := pbToPeerInfo(req.Info)
		if err != nil {
			log.Printf("directory/register: bad peer info from %s\n", pid.Pretty())
			break
		}

		if pinfo.ID != pid {
			log.Printf("directory/register: bogus peer info from %s\n", pid.Pretty())
			break
		}

		dir.registerPeer(pinfo)

		req.Reset()
	}

	dir.unregisterPeer(pid)
}

func (dir *Directory) lookupHandler(s p2p_net.Stream) {

}

func (dir *Directory) listHandler(s p2p_net.Stream) {

}

func (dir *Directory) registerPeer(info p2p_pstore.PeerInfo) {
	log.Printf("directory: register %s\n", info.ID.Pretty())
	dir.mx.Lock()
	dir.peers[info.ID] = info
	dir.mx.Unlock()
}

func (dir *Directory) unregisterPeer(pid p2p_peer.ID) {
	log.Printf("directory: unregister %s\n", pid.Pretty())
	dir.mx.Lock()
	delete(dir.peers, pid)
	dir.mx.Unlock()
}

func main() {
	log.Printf("Generating key pair\n")
	privk, pubk, err := p2p_crypto.GenerateKeyPair(p2p_crypto.RSA, 1024)
	if err != nil {
		log.Fatal(err)
	}

	id, err := p2p_peer.IDFromPublicKey(pubk)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("ID: %s\n", id.Pretty())

	pstore := p2p_pstore.NewPeerstore()
	pstore.AddPrivKey(id, privk)
	pstore.AddPubKey(id, pubk)

	addr, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/9000")
	if err != nil {
		log.Fatal(err)
	}

	netw, err := p2p_swarm.NewNetwork(
		context.Background(),
		[]multiaddr.Multiaddr{addr},
		id,
		pstore,
		p2p_metrics.NewBandwidthCounter())

	if err != nil {
		log.Fatal(err)
	}

	host := p2p_bhost.New(netw)
	dir := &Directory{pkey: privk, id: id, host: host}
	host.SetStreamHandler("/mediachain/dir/register", dir.registerHandler)
	host.SetStreamHandler("/mediachain/dir/lookup", dir.lookupHandler)
	host.SetStreamHandler("/mediachain/dir/list", dir.listHandler)

	log.Printf("I am %s/%s", addr, id.Pretty())
	select {}
}

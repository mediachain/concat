package main

import (
	"context"
	p2p_crypto "github.com/ipfs/go-libp2p-crypto"
	p2p_peer "github.com/ipfs/go-libp2p-peer"
	p2p_pstore "github.com/ipfs/go-libp2p-peerstore"
	multiaddr "github.com/jbenet/go-multiaddr"
	p2p_host "github.com/libp2p/go-libp2p/p2p/host"
	p2p_bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	p2p_metrics "github.com/libp2p/go-libp2p/p2p/metrics"
	p2p_net "github.com/libp2p/go-libp2p/p2p/net"
	p2p_swarm "github.com/libp2p/go-libp2p/p2p/net/swarm"
	"log"
	//	pb "github.com/mediachain/concat/proto"
)

type Directory struct {
	pkey  p2p_crypto.PrivKey
	id    p2p_peer.ID
	host  p2p_host.Host
	nodes map[p2p_peer.ID]p2p_pstore.PeerInfo
}

func (dir *Directory) registerHandler(s p2p_net.Stream) {

}

func (dir *Directory) lookupHandler(s p2p_net.Stream) {

}

func (dir *Directory) listHandler(s p2p_net.Stream) {

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

	host := p2p_bhost.New(netw)
	dir := &Directory{pkey: privk, id: id, host: host}
	host.SetStreamHandler("/mediachain/dir/register", dir.registerHandler)
	host.SetStreamHandler("/mediachain/dir/lookup", dir.lookupHandler)
	host.SetStreamHandler("/mediachain/dir/list", dir.listHandler)

	log.Printf("I am %s/%s", addr, id.Pretty())
	select {}
}

package main

import (
	"flag"
	"fmt"
	p2p_crypto "github.com/ipfs/go-libp2p-crypto"
	p2p_peer "github.com/ipfs/go-libp2p-peer"
	p2p_pstore "github.com/ipfs/go-libp2p-peerstore"
	multiaddr "github.com/jbenet/go-multiaddr"
	p2p_host "github.com/libp2p/go-libp2p/p2p/host"
	p2p_net "github.com/libp2p/go-libp2p/p2p/net"
	mc "github.com/mediachain/concat/mc"
	"log"
	"os"
)

type Node struct {
	id    p2p_peer.ID
	privk p2p_crypto.PrivKey
	host  p2p_host.Host
}

func (node *Node) pingHandler(s p2p_net.Stream) {

}

func (node *Node) registerPeer(dir p2p_pstore.PeerInfo, addr multiaddr.Multiaddr) {

}

func main() {
	port := flag.Int("l", 9001, "Listen port")
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Printf("Usage: %s [-l port] directory\n", os.Args[0])
		os.Exit(1)
	}

	addr, err := mc.ParseAddress(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", *port))
	if err != nil {
		log.Fatal(err)
	}

	dir, err := mc.ParseHandle(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Generating key pair\n")
	privk, pubk, err := mc.GenerateKeyPair()
	if err != nil {
		log.Fatal(err)
	}

	id, err := p2p_peer.IDFromPublicKey(pubk)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("ID: %s\n", id.Pretty())

	host, err := mc.NewHost(privk, addr)
	if err != nil {
		log.Fatal(err)
	}

	node := &Node{id: id, privk: privk, host: host}
	host.SetStreamHandler("/mediachain/node/ping", node.pingHandler)
	go node.registerPeer(dir, addr)

	log.Printf("I am %s/%s", addr, id.Pretty())
	select {}
}

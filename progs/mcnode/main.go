package main

import (
	"context"
	"flag"
	"fmt"
	ggio "github.com/gogo/protobuf/io"
	p2p_crypto "github.com/ipfs/go-libp2p-crypto"
	p2p_peer "github.com/ipfs/go-libp2p-peer"
	p2p_pstore "github.com/ipfs/go-libp2p-peerstore"
	multiaddr "github.com/jbenet/go-multiaddr"
	p2p_host "github.com/libp2p/go-libp2p/p2p/host"
	p2p_net "github.com/libp2p/go-libp2p/p2p/net"
	mc "github.com/mediachain/concat/mc"
	pb "github.com/mediachain/concat/proto"
	"log"
	"os"
	"time"
)

type Node struct {
	id    p2p_peer.ID
	privk p2p_crypto.PrivKey
	host  p2p_host.Host
}

func (node *Node) pingHandler(s p2p_net.Stream) {

}

func (node *Node) registerPeer(dir p2p_pstore.PeerInfo, addrs ...multiaddr.Multiaddr) {
	// directory failure is a fatality for now
	ctx := context.Background()

	err := node.host.Connect(ctx, dir)
	if err != nil {
		log.Printf("Failed to connect to directory")
		log.Fatal(err)
	}

	s, err := node.host.NewStream(ctx, dir.ID, "/mediachain/dir/register")
	if err != nil {
		log.Printf("Failed to open directory stream")
		log.Fatal(err)
	}
	defer s.Close()

	pbpi := pb.PeerInfo{}
	pbpi.Id = string(node.id)
	pbpi.Addr = make([][]byte, len(addrs))
	for x, addr := range addrs {
		pbpi.Addr[x] = addr.Bytes()
	}
	msg := pb.RegisterPeer{&pbpi}

	w := ggio.NewDelimitedWriter(s)
	for {
		log.Printf("registering with directory")
		err = w.WriteMsg(&msg)
		if err != nil {
			log.Printf("Failed to register with directory")
			log.Fatal(err)
		}

		time.Sleep(5 * time.Minute)
	}
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

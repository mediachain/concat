package main

import (
	"flag"
	"fmt"
	ggio "github.com/gogo/protobuf/io"
	p2p_crypto "github.com/ipfs/go-libp2p-crypto"
	p2p_peer "github.com/ipfs/go-libp2p-peer"
	p2p_pstore "github.com/ipfs/go-libp2p-peerstore"
	p2p_host "github.com/libp2p/go-libp2p/p2p/host"
	p2p_net "github.com/libp2p/go-libp2p/p2p/net"
	mc "github.com/mediachain/concat/mc"
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

func (dir *Directory) registerHandler(s p2p_net.Stream) {
	defer s.Close()

	pid := s.Conn().RemotePeer()
	log.Printf("directory/register: new stream from %s\n", pid.Pretty())

	reader := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
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

		pinfo, err := mc.PBToPeerInfo(req.Info)
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
	port := flag.Int("l", 9000, "Listen port")
	flag.Parse()

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

	addr, err := mc.ParseAddress(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", *port))
	if err != nil {
		log.Fatal(err)
	}

	host, err := mc.NewHost(privk, addr)
	if err != nil {
		log.Fatal(err)
	}

	dir := &Directory{pkey: privk, id: id, host: host, peers: make(map[p2p_peer.ID]p2p_pstore.PeerInfo)}
	host.SetStreamHandler("/mediachain/dir/register", dir.registerHandler)
	host.SetStreamHandler("/mediachain/dir/lookup", dir.lookupHandler)
	host.SetStreamHandler("/mediachain/dir/list", dir.listHandler)

	log.Printf("I am %s/%s", addr, id.Pretty())
	select {}
}

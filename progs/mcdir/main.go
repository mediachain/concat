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

	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	req := new(pb.RegisterPeer)

	for {
		err := r.ReadMsg(req)
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
	defer s.Close()

	pid := s.Conn().RemotePeer()
	log.Printf("directory/lookup: new stream from %s\n", pid.Pretty())

	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	w := ggio.NewDelimitedWriter(s)
	req := new(pb.LookupPeerRequest)
	resp := new(pb.LookupPeerResponse)

	for {
		err := r.ReadMsg(req)
		if err != nil {
			break
		}

		pid, err := p2p_peer.IDFromString(req.Id)
		if err != nil {
			log.Printf("directory/lookup: bad request from %s\n", pid.Pretty())
			break
		}

		pinfo, ok := dir.lookupPeer(pid)
		if ok {
			var pbpi pb.PeerInfo
			mc.PBFromPeerInfo(&pbpi, pinfo)
			resp.Peer = &pbpi
		}

		err = w.WriteMsg(resp)
		if err != nil {
			break
		}

		req.Reset()
		resp.Reset()
	}
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

func (dir *Directory) lookupPeer(pid p2p_peer.ID) (p2p_pstore.PeerInfo, bool) {
	dir.mx.Lock()
	pinfo, ok := dir.peers[pid]
	dir.mx.Unlock()
	return pinfo, ok
}

func main() {
	port := flag.Int("l", 9000, "Listen port")
	home := flag.String("d", "/tmp/mcdir", "Directory home")
	flag.Parse()

	mc.EnsureDirectory(*home, 0755)
	id, err := mc.NodeIdentity(*home)
	if err != nil {
		log.Fatal(err)
	}

	addr, err := mc.ParseAddress(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", *port))
	if err != nil {
		log.Fatal(err)
	}

	host, err := mc.NewHost(id, addr)
	if err != nil {
		log.Fatal(err)
	}

	dir := &Directory{pkey: id.PrivKey, id: id.ID, host: host, peers: make(map[p2p_peer.ID]p2p_pstore.PeerInfo)}
	host.SetStreamHandler("/mediachain/dir/register", dir.registerHandler)
	host.SetStreamHandler("/mediachain/dir/lookup", dir.lookupHandler)
	host.SetStreamHandler("/mediachain/dir/list", dir.listHandler)

	log.Printf("I am %s/%s", addr, id.Pretty())
	select {}
}

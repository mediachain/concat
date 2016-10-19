package main

import (
	"flag"
	"fmt"
	ggio "github.com/gogo/protobuf/io"
	p2p_host "github.com/libp2p/go-libp2p-host"
	p2p_net "github.com/libp2p/go-libp2p-net"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	p2p_pstore "github.com/libp2p/go-libp2p-peerstore"
	mc "github.com/mediachain/concat/mc"
	pb "github.com/mediachain/concat/proto"
	homedir "github.com/mitchellh/go-homedir"
	"log"
	"os"
	"sync"
)

type Directory struct {
	mc.PeerIdentity
	host  p2p_host.Host
	peers map[p2p_peer.ID]p2p_pstore.PeerInfo
	mx    sync.Mutex
}

func (dir *Directory) registerHandler(s p2p_net.Stream) {
	defer s.Close()

	pid := s.Conn().RemotePeer()
	paddr := s.Conn().RemoteMultiaddr()
	log.Printf("directory/register: new stream from %s at %s", pid.Pretty(), paddr.String())

	var req pb.RegisterPeer
	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)

	for {
		err := r.ReadMsg(&req)
		if err != nil {
			break
		}

		if req.Info == nil {
			log.Printf("directory/register: empty peer info from %s", pid.Pretty())
			break
		}

		pinfo, err := mc.PBToPeerInfo(req.Info)
		if err != nil {
			log.Printf("directory/register: bad peer info from %s", pid.Pretty())
			break
		}

		if pinfo.ID != pid {
			log.Printf("directory/register: bogus peer info from %s", pid.Pretty())
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
	paddr := s.Conn().RemoteMultiaddr()
	log.Printf("directory/lookup: new stream from %s at %s", pid.Pretty(), paddr.String())

	var req pb.LookupPeerRequest
	var res pb.LookupPeerResponse

	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	w := ggio.NewDelimitedWriter(s)

	for {
		err := r.ReadMsg(&req)
		if err != nil {
			break
		}

		xid, err := p2p_peer.IDB58Decode(req.Id)
		if err != nil {
			log.Printf("directory/lookup: bad request from %s", pid.Pretty())
			break
		}

		pinfo, ok := dir.lookupPeer(xid)
		if ok {
			var pbpi pb.PeerInfo
			mc.PBFromPeerInfo(&pbpi, pinfo)
			res.Peer = &pbpi
		}

		err = w.WriteMsg(&res)
		if err != nil {
			break
		}

		req.Reset()
		res.Reset()
	}
}

func (dir *Directory) listHandler(s p2p_net.Stream) {
	defer s.Close()

	pid := s.Conn().RemotePeer()
	paddr := s.Conn().RemoteMultiaddr()
	log.Printf("directory/list: new stream from %s at %s", pid.Pretty(), paddr.String())

	var req pb.ListPeersRequest
	var res pb.ListPeersResponse

	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	w := ggio.NewDelimitedWriter(s)

	for {
		err := r.ReadMsg(&req)
		if err != nil {
			break
		}

		res.Peers = dir.listPeers()

		err = w.WriteMsg(&res)
		if err != nil {
			break
		}

		res.Reset()
	}
}

func (dir *Directory) registerPeer(info p2p_pstore.PeerInfo) {
	log.Printf("directory: register %s", info.ID.Pretty())
	dir.mx.Lock()
	dir.peers[info.ID] = info
	dir.mx.Unlock()
}

func (dir *Directory) unregisterPeer(pid p2p_peer.ID) {
	log.Printf("directory: unregister %s", pid.Pretty())
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

func (dir *Directory) listPeers() []string {
	dir.mx.Lock()
	lst := make([]string, len(dir.peers))
	for pid, _ := range dir.peers {
		lst = append(lst, pid.Pretty())
	}
	dir.mx.Unlock()
	return lst
}

func main() {
	port := flag.Int("l", 9000, "Listen port")
	hdir := flag.String("d", "~/.mediachain/mcdir", "Directory home")
	flag.Parse()

	if len(flag.Args()) != 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options ...]\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	home, err := homedir.Expand(*hdir)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(home, 0755)
	if err != nil {
		log.Fatal(err)
	}

	id, err := mc.MakePeerIdentity(home)
	if err != nil {
		log.Fatal(err)
	}

	addr, err := mc.ParseAddress(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port))
	if err != nil {
		log.Fatal(err)
	}

	host, err := mc.NewHost(id, addr)
	if err != nil {
		log.Fatal(err)
	}

	dir := &Directory{PeerIdentity: id, host: host, peers: make(map[p2p_peer.ID]p2p_pstore.PeerInfo)}
	host.SetStreamHandler("/mediachain/dir/register", dir.registerHandler)
	host.SetStreamHandler("/mediachain/dir/lookup", dir.lookupHandler)
	host.SetStreamHandler("/mediachain/dir/list", dir.listHandler)

	for _, addr := range host.Addrs() {
		if !mc.IsLinkLocalAddr(addr) {
			log.Printf("I am %s/%s", addr, id.Pretty())
		}
	}
	select {}
}

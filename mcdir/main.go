package main

import (
	"context"
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
	"strings"
	"sync"
)

type Directory struct {
	mc.PeerIdentity
	host  p2p_host.Host
	peers map[p2p_peer.ID]PeerRecord
	mx    sync.Mutex
}

type PeerRecord struct {
	peer      p2p_pstore.PeerInfo
	publisher *pb.PublisherInfo
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

		dir.registerPeer(PeerRecord{pinfo, req.Publisher})

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

		res.Peers = dir.listPeers(req.Namespace)

		err = w.WriteMsg(&res)
		if err != nil {
			break
		}

		res.Reset()
	}
}

func (dir *Directory) registerPeer(rec PeerRecord) {
	log.Printf("directory: register %s", rec.peer.ID.Pretty())
	dir.mx.Lock()
	dir.peers[rec.peer.ID] = rec
	dir.mx.Unlock()
}

func (dir *Directory) unregisterPeer(pid p2p_peer.ID) {
	log.Printf("directory: unregister %s", pid.Pretty())
	dir.mx.Lock()
	delete(dir.peers, pid)
	dir.mx.Unlock()
}

func (dir *Directory) lookupPeer(pid p2p_peer.ID) (p2p_pstore.PeerInfo, bool) {
	log.Printf("directory: lookup %s", pid.Pretty())
	dir.mx.Lock()
	rec, ok := dir.peers[pid]
	dir.mx.Unlock()
	return rec.peer, ok
}

func (dir *Directory) listPeers(ns string) []string {
	switch {
	case ns == "":
		fallthrough
	case ns == "*":
		return dir.listPeersFilter(func(PeerRecord) bool {
			return true
		})

	case strings.HasSuffix(ns, ".*"):
		pre := ns[:len(ns)-2]
		return dir.listPeersFilter(func(rec PeerRecord) bool {
			if rec.publisher == nil {
				return false
			}

			for _, xns := range rec.publisher.Namespaces {
				if strings.HasPrefix(xns, pre) {
					return true
				}
			}

			return false
		})

	default:
		return dir.listPeersFilter(func(rec PeerRecord) bool {
			if rec.publisher == nil {
				return false
			}

			for _, xns := range rec.publisher.Namespaces {
				if ns == xns {
					return true
				}
			}

			return false
		})

	}
}

func (dir *Directory) listPeersFilter(filter func(PeerRecord) bool) []string {
	dir.mx.Lock()
	lst := make([]string, 0, len(dir.peers))
	for pid, rec := range dir.peers {
		if filter(rec) {
			lst = append(lst, pid.Pretty())
		}
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

	host, err := mc.NewHost(context.Background(), id, addr)
	if err != nil {
		log.Fatal(err)
	}

	dir := &Directory{PeerIdentity: id, host: host, peers: make(map[p2p_peer.ID]PeerRecord)}
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

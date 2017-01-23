package main

import (
	p2p_host "github.com/libp2p/go-libp2p-host"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	p2p_pstore "github.com/libp2p/go-libp2p-peerstore"
	mc "github.com/mediachain/concat/mc"
	pb "github.com/mediachain/concat/proto"
	"log"
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
	log.Printf("directory: list %s", ns)

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

func (dir *Directory) listNamespaces() []string {
	log.Printf("directory: listns")

	nsset := make(map[string]bool)

	dir.mx.Lock()
	for _, rec := range dir.peers {
		if rec.publisher != nil {
			for _, ns := range rec.publisher.Namespaces {
				nsset[ns] = true
			}
		}
	}
	dir.mx.Unlock()

	nslst := make([]string, 0, len(nsset))
	for ns, _ := range nsset {
		nslst = append(nslst, ns)
	}

	return nslst
}

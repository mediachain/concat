package main

import (
	ggio "github.com/gogo/protobuf/io"
	p2p_net "github.com/libp2p/go-libp2p-net"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	mc "github.com/mediachain/concat/mc"
	pb "github.com/mediachain/concat/proto"
	"log"
)

func (dir *Directory) registerHandler(s p2p_net.Stream) {
	defer s.Close()

	pid := mc.LogStreamHandler(s)

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

		if len(req.Manifest) > 0 {
			dir.mfs.Put(pid, req.Manifest)
		}

		req.Reset()
	}

	dir.unregisterPeer(pid)
	dir.mfs.Remove(pid)
}

func (dir *Directory) lookupHandler(s p2p_net.Stream) {
	defer s.Close()

	pid := mc.LogStreamHandler(s)

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

	mc.LogStreamHandler(s)

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

func (dir *Directory) listnsHandler(s p2p_net.Stream) {
	defer s.Close()

	mc.LogStreamHandler(s)

	var req pb.ListNamespacesRequest
	var res pb.ListNamespacesResponse

	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	w := ggio.NewDelimitedWriter(s)

	for {
		err := r.ReadMsg(&req)
		if err != nil {
			break
		}

		res.Namespaces = dir.listNamespaces()

		err = w.WriteMsg(&res)
		if err != nil {
			break
		}

		res.Reset()
	}
}

func (dir *Directory) listmfHandler(s p2p_net.Stream) {
	defer s.Close()

	mc.LogStreamHandler(s)

	var req pb.ListManifestRequest
	var res pb.ListManifestResponse

	r := ggio.NewDelimitedReader(s, mc.MaxMessageSize)
	w := ggio.NewDelimitedWriter(s)

	for {
		err := r.ReadMsg(&req)
		if err != nil {
			break
		}

		res.Manifest = dir.mfs.Lookup(req.Entity)

		err = w.WriteMsg(&res)
		if err != nil {
			break
		}

		res.Reset()
	}
}

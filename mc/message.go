package mc

import (
	p2p_peer "github.com/ipfs/go-libp2p-peer"
	p2p_pstore "github.com/ipfs/go-libp2p-peerstore"
	multiaddr "github.com/jbenet/go-multiaddr"
	pb "github.com/mediachain/concat/proto"
)

const MaxMessageSize = 2 << 20 // 1 MB

func PBFromPeerInfo(pbpi *pb.PeerInfo, pinfo p2p_pstore.PeerInfo) {
	pbpi.Id = string(pinfo.ID)
	pbpi.Addr = make([][]byte, len(pinfo.Addrs))
	for x, addr := range pinfo.Addrs {
		pbpi.Addr[x] = addr.Bytes()
	}
}

func PBToPeerInfo(pbpi *pb.PeerInfo) (empty p2p_pstore.PeerInfo, err error) {
	pid := p2p_peer.ID(pbpi.Id)
	addrs := make([]multiaddr.Multiaddr, len(pbpi.Addr))
	for x, bytes := range pbpi.Addr {
		addr, err := multiaddr.NewMultiaddrBytes(bytes)
		if err != nil {
			return empty, err
		}
		addrs[x] = addr
	}

	return p2p_pstore.PeerInfo{pid, addrs}, nil
}

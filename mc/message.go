package mc

import (
	"fmt"
	p2p_peer "github.com/ipfs/go-libp2p-peer"
	p2p_pstore "github.com/ipfs/go-libp2p-peerstore"
	multiaddr "github.com/jbenet/go-multiaddr"
	pb "github.com/mediachain/concat/proto"
)

const MaxMessageSize = 2 << 20 // 1 MB

func PBFromPeerInfo(pbpi *pb.PeerInfo, pinfo p2p_pstore.PeerInfo) {
	pbpi.Id = pinfo.ID.Pretty()
	pbpi.Addr = make([][]byte, len(pinfo.Addrs))
	for x, addr := range pinfo.Addrs {
		pbpi.Addr[x] = addr.Bytes()
	}
}

func PBToPeerInfo(pbpi *pb.PeerInfo) (empty p2p_pstore.PeerInfo, err error) {
	pid, err := p2p_peer.IDB58Decode(pbpi.Id)
	if err != nil {
		return empty, err
	}
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

type ValueError string

func (v ValueError) Error() string {
	return string(v)
}

func SimpleValue(val interface{}) (*pb.SimpleValue, error) {
	switch val := val.(type) {
	case int:
		return &pb.SimpleValue{&pb.SimpleValue_IntValue{int64(val)}}, nil

	case int64:
		return &pb.SimpleValue{&pb.SimpleValue_IntValue{val}}, nil

	case string:
		return &pb.SimpleValue{&pb.SimpleValue_StringValue{val}}, nil

	case *pb.Statement:
		return &pb.SimpleValue{&pb.SimpleValue_Stmt{val}}, nil

	case *pb.StatementBody:
		return &pb.SimpleValue{&pb.SimpleValue_StmtBody{val}}, nil

	default:
		return nil, ValueError(fmt.Sprintf("Unexpected value type: %T", val))
	}
}

func CompoundValue(val map[string]interface{}) (*pb.CompoundValue, error) {
	kvpairs := make([]*pb.KeyValuePair, 0, len(val))
	for k, v := range val {
		sv, err := SimpleValue(v)
		if err != nil {
			return nil, err
		}
		kvpairs = append(kvpairs, &pb.KeyValuePair{k, sv})
	}
	return &pb.CompoundValue{kvpairs}, nil
}

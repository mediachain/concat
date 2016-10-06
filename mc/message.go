package mc

import (
	"fmt"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	p2p_pstore "github.com/libp2p/go-libp2p-peerstore"
	pb "github.com/mediachain/concat/proto"
	multiaddr "github.com/multiformats/go-multiaddr"
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

func ValueOf(res *pb.QueryResultValue) (interface{}, error) {
	switch val := res.Value.(type) {
	case *pb.QueryResultValue_Simple:
		return SimpleValueOf(val.Simple)

	case *pb.QueryResultValue_Compound:
		obj := make(map[string]interface{})
		for _, kv := range val.Compound.Body {
			sv, err := SimpleValueOf(kv.Value)
			if err != nil {
				return nil, err
			}

			obj[kv.Key] = sv
		}

		return obj, nil

	default:
		return nil, ValueError(fmt.Sprintf("Unexpected value type: %T", val))
	}
}

func SimpleValueOf(sv *pb.SimpleValue) (interface{}, error) {
	switch val := sv.Value.(type) {
	case *pb.SimpleValue_IntValue:
		return val.IntValue, nil

	case *pb.SimpleValue_StringValue:
		return val.StringValue, nil

	case *pb.SimpleValue_Stmt:
		return val.Stmt, nil

	case *pb.SimpleValue_StmtBody:
		return val.StmtBody, nil

	default:
		return nil, ValueError(fmt.Sprintf("Unexpected value type: %T", val))
	}
}

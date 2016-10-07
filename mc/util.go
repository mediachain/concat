package mc

import (
	"context"
	"errors"
	"fmt"
	p2p_host "github.com/libp2p/go-libp2p-host"
	p2p_metrics "github.com/libp2p/go-libp2p-metrics"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	p2p_pstore "github.com/libp2p/go-libp2p-peerstore"
	p2p_swarm "github.com/libp2p/go-libp2p-swarm"
	p2p_bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	multiaddr "github.com/multiformats/go-multiaddr"
	"strings"
)

func ParseAddress(str string) (multiaddr.Multiaddr, error) {
	return multiaddr.NewMultiaddr(str)
}

var BadHandle = errors.New("Bad handle")

// handle: multiaddr/id
func FormatHandle(pi p2p_pstore.PeerInfo) string {
	if len(pi.Addrs) > 0 {
		return fmt.Sprintf("%s/%s", pi.Addrs[0].String(), pi.ID.Pretty())
	} else {
		return pi.ID.Pretty()
	}
}

func ParseHandle(str string) (empty p2p_pstore.PeerInfo, err error) {
	ix := strings.LastIndex(str, "/")
	if ix < 0 {
		return ParseHandleId(str)
	}

	addr, id := str[:ix], str[ix+1:]

	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return empty, err
	}

	pid, err := p2p_peer.IDB58Decode(id)
	if err != nil {
		return empty, err
	}

	return p2p_pstore.PeerInfo{ID: pid, Addrs: []multiaddr.Multiaddr{maddr}}, nil
}

func ParseHandleId(str string) (empty p2p_pstore.PeerInfo, err error) {
	pid, err := p2p_peer.IDB58Decode(str)
	if err != nil {
		return empty, err
	}

	return p2p_pstore.PeerInfo{ID: pid}, nil
}

// re-export this option to avoid basic host interface leakage
const NATPortMap = p2p_bhost.NATPortMap

func NewHost(id PeerIdentity, laddr multiaddr.Multiaddr, opts ...interface{}) (p2p_host.Host, error) {
	pstore := p2p_pstore.NewPeerstore()
	pstore.AddPrivKey(id.ID, id.PrivKey)
	pstore.AddPubKey(id.ID, id.PrivKey.GetPublic())

	netw, err := p2p_swarm.NewNetwork(
		context.Background(),
		[]multiaddr.Multiaddr{laddr},
		id.ID,
		pstore,
		p2p_metrics.NewBandwidthCounter())
	if err != nil {
		return nil, err
	}

	return p2p_bhost.New(netw, opts...), nil
}

// multiaddr juggling
func isAddrSubnet(addr multiaddr.Multiaddr, prefix []string) bool {
	ip, err := addr.ValueForProtocol(multiaddr.P_IP4)
	if err != nil {
		return false
	}

	for _, pre := range prefix {
		if strings.HasPrefix(ip, pre) {
			return true
		}
	}

	return false
}

var (
	localhostSubnet = []string{"127."}
	linkLocalSubnet = []string{"169.254."}
	privateSubnet   = []string{"10.", "172.16.", "192.168."}
	internalSubnet  = []string{"127.", "169.254.", "10.", "172.16.", "192.168."}
)

func IsLocalhostAddr(addr multiaddr.Multiaddr) bool {
	return isAddrSubnet(addr, localhostSubnet)
}

func IsLinkLocalAddr(addr multiaddr.Multiaddr) bool {
	return isAddrSubnet(addr, linkLocalSubnet)
}

func IsPrivateAddr(addr multiaddr.Multiaddr) bool {
	return isAddrSubnet(addr, privateSubnet)
}

func IsPublicAddr(addr multiaddr.Multiaddr) bool {
	return !isAddrSubnet(addr, internalSubnet)
}

func FilterAddrs(addrs []multiaddr.Multiaddr, predf func(multiaddr.Multiaddr) bool) []multiaddr.Multiaddr {
	res := make([]multiaddr.Multiaddr, 0, len(addrs))
	for _, addr := range addrs {
		if predf(addr) {
			res = append(res, addr)
		}
	}
	return res
}

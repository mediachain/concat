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
	multihash "github.com/multiformats/go-multihash"
	"log"
	"net"
	"strings"
)

func Hash(data []byte) multihash.Multihash {
	// this can only fail with an error because of an incorrect hash family
	mh, _ := multihash.Sum(data, multihash.SHA2_256, -1)
	return mh
}

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
func isAddrSubnet(addr multiaddr.Multiaddr, nets []*net.IPNet) bool {
	ipstr, err := addr.ValueForProtocol(multiaddr.P_IP4)
	if err != nil {
		return false
	}

	ip := net.ParseIP(ipstr)
	if ip == nil {
		return false
	}

	for _, net := range nets {
		if net.Contains(ip) {
			return true
		}
	}

	return false
}

var (
	localhostCIDR  = []string{"127.0.0.0/8"}
	linkLocalCIDR  = []string{"169.254.0.0/16"}
	privateCIDR    = []string{"10.0.0.0/8", "100.64.0.0/10", "172.16.0.0/12", "192.168.0.0/16"}
	unroutableCIDR = []string{"0.0.0.0/8", "127.0.0.0/8", "169.254.0.0/16"}
	internalCIDR   = []string{"0.0.0.0/8", "127.0.0.0/8", "169.254.0.0/16", "10.0.0.0/8", "100.64.0.0/10", "172.16.0.0/12", "192.168.0.0/16"}
)

var (
	localhostSubnet  []*net.IPNet
	linkLocalSubnet  []*net.IPNet
	privateSubnet    []*net.IPNet
	unroutableSubnet []*net.IPNet
	internalSubnet   []*net.IPNet
)

func makeSubnetSpec(subnets []string) []*net.IPNet {
	nets := make([]*net.IPNet, len(subnets))
	for x, subnet := range subnets {
		_, net, err := net.ParseCIDR(subnet)
		if err != nil {
			log.Fatal(err)
		}
		nets[x] = net
	}
	return nets
}

func init() {
	localhostSubnet = makeSubnetSpec(localhostCIDR)
	linkLocalSubnet = makeSubnetSpec(linkLocalCIDR)
	privateSubnet = makeSubnetSpec(privateCIDR)
	unroutableSubnet = makeSubnetSpec(unroutableCIDR)
	internalSubnet = makeSubnetSpec(internalCIDR)
}

func IsLocalhostAddr(addr multiaddr.Multiaddr) bool {
	return isAddrSubnet(addr, localhostSubnet)
}

func IsLinkLocalAddr(addr multiaddr.Multiaddr) bool {
	return isAddrSubnet(addr, linkLocalSubnet)
}

func IsPrivateAddr(addr multiaddr.Multiaddr) bool {
	return isAddrSubnet(addr, privateSubnet)
}

func IsRoutableAddr(addr multiaddr.Multiaddr) bool {
	return !isAddrSubnet(addr, unroutableSubnet)
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

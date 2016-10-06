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

func NewHost(id NodeIdentity, addrs ...multiaddr.Multiaddr) (p2p_host.Host, error) {
	pstore := p2p_pstore.NewPeerstore()
	pstore.AddPrivKey(id.ID, id.PrivKey)
	pstore.AddPubKey(id.ID, id.PrivKey.GetPublic())

	netw, err := p2p_swarm.NewNetwork(
		context.Background(),
		addrs,
		id.ID,
		pstore,
		p2p_metrics.NewBandwidthCounter())
	if err != nil {
		return nil, err
	}

	return p2p_bhost.New(netw), nil
}

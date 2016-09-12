package mc

import (
	"context"
	"errors"
	p2p_crypto "github.com/ipfs/go-libp2p-crypto"
	p2p_peer "github.com/ipfs/go-libp2p-peer"
	p2p_pstore "github.com/ipfs/go-libp2p-peerstore"
	multiaddr "github.com/jbenet/go-multiaddr"
	p2p_host "github.com/libp2p/go-libp2p/p2p/host"
	p2p_bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	p2p_metrics "github.com/libp2p/go-libp2p/p2p/metrics"
	p2p_swarm "github.com/libp2p/go-libp2p/p2p/net/swarm"
	"strings"
)

func ParseAddress(str string) (multiaddr.Multiaddr, error) {
	return multiaddr.NewMultiaddr(str)
}

var BadHandle = errors.New("Bad handle")

// handle: multiaddr/id
func ParseHandle(str string) (empty p2p_pstore.PeerInfo, err error) {
	ix := strings.LastIndex(str, "/")
	if ix < 0 {
		return empty, BadHandle
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

func GenerateKeyPair() (p2p_crypto.PrivKey, p2p_crypto.PubKey, error) {
	// 1kbit RSA just for now -- until ECC key support is in mrged into libp2p
	return p2p_crypto.GenerateKeyPair(p2p_crypto.RSA, 1024)
}

func NewHost(privk p2p_crypto.PrivKey, addrs ...multiaddr.Multiaddr) (p2p_host.Host, error) {
	pstore := p2p_pstore.NewPeerstore()
	pubk := privk.GetPublic()
	id, err := p2p_peer.IDFromPublicKey(pubk)
	if err != nil {
		return nil, err
	}

	pstore.AddPrivKey(id, privk)
	pstore.AddPubKey(id, pubk)

	netw, err := p2p_swarm.NewNetwork(
		context.Background(),
		addrs,
		id,
		pstore,
		p2p_metrics.NewBandwidthCounter())
	if err != nil {
		return nil, err
	}

	return p2p_bhost.New(netw), nil
}

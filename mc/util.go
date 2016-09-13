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
	"io/ioutil"
	"log"
	"os"
	"path"
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

// node identities
type Identity struct {
	ID      p2p_peer.ID
	PrivKey p2p_crypto.PrivKey
}

func (id Identity) Pretty() string {
	return id.ID.Pretty()
}

func NodeIdentity(home string) (empty Identity, err error) {
	kpath := path.Join(home, "identity")
	_, err = os.Stat(kpath)
	if os.IsNotExist(err) {
		return generateIdentity(kpath)
	}
	if err != nil {
		return
	}
	return loadIdentity(kpath)
}

func generateIdentity(kpath string) (empty Identity, err error) {
	log.Printf("Generating new identity")
	privk, pubk, err := generateKeyPair()
	if err != nil {
		return
	}

	id, err := p2p_peer.IDFromPublicKey(pubk)
	if err != nil {
		return
	}

	log.Printf("Saving identity to %s", kpath)
	bytes, err := privk.Bytes()
	if err != nil {
		return
	}

	err = ioutil.WriteFile(kpath, bytes, 0600)
	if err != nil {
		return
	}

	log.Printf("ID: %s", id.Pretty())
	return Identity{ID: id, PrivKey: privk}, nil
}

func loadIdentity(kpath string) (empty Identity, err error) {
	log.Printf("Loading identity from %s", kpath)
	bytes, err := ioutil.ReadFile(kpath)
	if err != nil {
		return
	}

	privk, err := p2p_crypto.UnmarshalPrivateKey(bytes)
	if err != nil {
		return
	}

	id, err := p2p_peer.IDFromPrivateKey(privk)
	if err != nil {
		return
	}

	log.Printf("ID: %s", id.Pretty())
	return Identity{ID: id, PrivKey: privk}, nil
}

func generateKeyPair() (p2p_crypto.PrivKey, p2p_crypto.PubKey, error) {
	// 1kbit RSA just for now -- until ECC key support is merged in libp2p
	return p2p_crypto.GenerateKeyPair(p2p_crypto.RSA, 1024)
}

func NewHost(id Identity, addrs ...multiaddr.Multiaddr) (p2p_host.Host, error) {
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

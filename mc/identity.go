package mc

import (
	p2p_crypto "github.com/libp2p/go-libp2p-crypto"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	"io/ioutil"
	"log"
	"os"
	"path"
)

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

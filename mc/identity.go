package mc

import (
	b58 "github.com/jbenet/go-base58"
	p2p_crypto "github.com/libp2p/go-libp2p-crypto"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	"io/ioutil"
	"log"
	"os"
	"path"
)

// Identities: PeerIdentity and PublisherIdentity
// the structs are different because the semantics of id differ
// PeerIdentities use the raw public key hash as dictated by libp2p
//  they use RSA keys for interop with js, which is lagging in libp2p-crypto
// PublisherIdentities use the base58 encoded public key as identifier
//  they use ECC (Ed25519) keys and sign statements with them
type PeerIdentity struct {
	ID      p2p_peer.ID
	PrivKey p2p_crypto.PrivKey
}

type PublisherIdentity struct {
	ID58    string
	PrivKey p2p_crypto.PrivKey
}

func (id PeerIdentity) Pretty() string {
	return id.ID.Pretty()
}

func (id PublisherIdentity) Pretty() string {
	return id.ID58
}

// Node Identities
func MakePeerIdentity(home string) (empty PeerIdentity, err error) {
	kpath := path.Join(home, "identity.node")
	_, err = os.Stat(kpath)
	if os.IsNotExist(err) {
		return generatePeerIdentity(kpath)
	}
	if err != nil {
		return
	}
	return loadPeerIdentity(kpath)
}

func generatePeerIdentity(kpath string) (empty PeerIdentity, err error) {
	log.Printf("Generating new node identity")
	// RSA keys for interop with js
	privk, pubk, err := generateRSAKeyPair()
	if err != nil {
		return
	}

	id, err := p2p_peer.IDFromPublicKey(pubk)
	if err != nil {
		return
	}

	log.Printf("Saving key to %s", kpath)
	err = saveKey(privk, kpath)
	if err != nil {
		return
	}

	log.Printf("Node ID: %s", id.Pretty())
	return PeerIdentity{ID: id, PrivKey: privk}, nil
}

func loadPeerIdentity(kpath string) (empty PeerIdentity, err error) {
	log.Printf("Loading node identity from %s", kpath)
	privk, err := loadKey(kpath)
	if err != nil {
		return
	}

	id, err := p2p_peer.IDFromPrivateKey(privk)
	if err != nil {
		return
	}

	log.Printf("Node ID: %s", id.Pretty())
	return PeerIdentity{id, privk}, nil
}

// Publisher Identities
func MakePublisherIdentity(home string) (empty PublisherIdentity, err error) {
	kpath := path.Join(home, "identity.publisher") // .pub would be unfortunate
	_, err = os.Stat(kpath)
	if os.IsNotExist(err) {
		return generatePublisherIdentity(kpath)
	}
	if err != nil {
		return
	}
	return loadPublisherIdentity(kpath)
}

func generatePublisherIdentity(kpath string) (empty PublisherIdentity, err error) {
	log.Printf("Generating new publisher identity")

	privk, pubk, err := generateECCKeyPair()
	if err != nil {
		return
	}

	id58, err := PublisherID58(pubk)
	if err != nil {
		return
	}

	log.Printf("Saving key to %s", kpath)
	err = saveKey(privk, kpath)
	if err != nil {
		return
	}

	log.Printf("Publisher ID: %s", id58)
	return PublisherIdentity{id58, privk}, nil

}

func loadPublisherIdentity(kpath string) (empty PublisherIdentity, err error) {
	log.Printf("Loading publisher identity from %s", kpath)
	privk, err := loadKey(kpath)
	if err != nil {
		return
	}

	id58, err := PublisherID58(privk.GetPublic())
	if err != nil {
		return
	}

	log.Printf("Publisher ID: %s", id58)
	return PublisherIdentity{id58, privk}, nil
}

func PublisherID58(pubk p2p_crypto.PubKey) (string, error) {
	bytes, err := pubk.Bytes()
	if err != nil {
		return "", err
	}

	id58 := b58.Encode(bytes)
	return id58, nil
}

func PublisherKey(id58 string) (p2p_crypto.PubKey, error) {
	bytes := b58.Decode(id58)
	return p2p_crypto.UnmarshalPublicKey(bytes)
}

// Key management
func loadKey(kpath string) (p2p_crypto.PrivKey, error) {
	bytes, err := ioutil.ReadFile(kpath)
	if err != nil {
		return nil, err
	}

	return p2p_crypto.UnmarshalPrivateKey(bytes)
}

func saveKey(privk p2p_crypto.PrivKey, kpath string) error {
	bytes, err := privk.Bytes()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(kpath, bytes, 0600)
}

func generateRSAKeyPair() (p2p_crypto.PrivKey, p2p_crypto.PubKey, error) {
	return p2p_crypto.GenerateKeyPair(p2p_crypto.RSA, 2048)
}

func generateECCKeyPair() (p2p_crypto.PrivKey, p2p_crypto.PubKey, error) {
	return p2p_crypto.GenerateKeyPair(p2p_crypto.Ed25519, 0)
}

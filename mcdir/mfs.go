package main

import (
	p2p_crypto "github.com/libp2p/go-libp2p-crypto"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	mc "github.com/mediachain/concat/mc"
	pb "github.com/mediachain/concat/proto"
	multihash "github.com/multiformats/go-multihash"
	"log"
	"sync"
)

type ManifestStoreImpl struct {
	mx   sync.Mutex
	mf   map[string]ManifestRecord
	keys map[string]p2p_crypto.PubKey
}

type ManifestRecord struct {
	mf  *pb.Manifest
	src p2p_peer.ID
}

func NewManifestStore() ManifestStore {
	return &ManifestStoreImpl{
		mf:   make(map[string]ManifestRecord),
		keys: make(map[string]p2p_crypto.PubKey),
	}
}

func (mfs *ManifestStoreImpl) Put(src p2p_peer.ID, lst []*pb.Manifest) {
	mfs.mx.Lock()
	defer mfs.mx.Unlock()

	for _, mf := range lst {
		mfs.putManifest(src, mf)
	}
}

func (mfs *ManifestStoreImpl) putManifest(src p2p_peer.ID, mf *pb.Manifest) {
	mfh := hashManifest(mf).B58String()

	_, ok := mfs.mf[mfh]
	if ok {
		return
	}

	kid := mf.Entity + ":" + mf.KeyId
	pubk, ok := mfs.keys[kid]

	if !ok {
		var err error
		// XXX This can take arbitrary long and might need to be throttled
		// XXX and in the meantime, it holds the manifest store lock...
		// XXX The solution to this problem is to fetch keys asynchronously
		// XXX with back-off/retry logic; impl is complicated though, so for now
		// XXX damn the torpedoes.
		pubk, err = mc.LookupEntityKey(mf.Entity, mf.KeyId)
		if err != nil {
			log.Printf("Error looking up entity key %s: %s", kid, err.Error())
			return
		}
		mfs.keys[kid] = pubk
	}

	ok, err := verifyManifest(mf, pubk)
	switch {
	case ok:
		// yay! a valid manifest.
		mfs.mf[mfh] = ManifestRecord{mf, src}

	case err != nil:
		log.Printf("Error verifying manifest %s: %s", mfh, err.Error())

	default:
		log.Printf("Error verifying manifest %s: signature verification failed", mfh)
	}
}

func (mfs *ManifestStoreImpl) Remove(src p2p_peer.ID) {
	// XXX Implement me
}

func (mfs *ManifestStoreImpl) Lookup(entity string) []*pb.Manifest {
	// XXX Implement me
	return nil
}

func hashManifest(mf *pb.Manifest) multihash.Multihash {
	// XXX Implement me
	return nil
}

func verifyManifest(mf *pb.Manifest, pubk p2p_crypto.PubKey) (bool, error) {
	// XXX Implement me
	return false, nil
}

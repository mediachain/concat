package main

import (
	ggproto "github.com/gogo/protobuf/proto"
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
	mfx, err := hashManifest(mf)
	if err != nil {
		log.Printf("Error hashing manifest; wtf: %s", err.Error())
		return
	}

	mfh := mfx.B58String()

	_, ok := mfs.mf[mfh]
	if ok {
		return
	}

	kid := mf.Entity + ":" + mf.KeyId
	pubk, ok := mfs.keys[kid]

	if !ok {
		// XXX This can take arbitrary long and might need to be throttled
		// XXX and in the meantime, it holds the manifest store lock...
		// XXX The solution to this problem is to fetch keys asynchronously
		// XXX with back-off/retry logic; impl is complicated though, so for now
		// XXX damn the torpedoes (but at least release the lock)
		mfs.mx.Unlock()
		pubk, err = mc.LookupEntityKey(mf.Entity, mf.KeyId)
		mfs.mx.Lock()
		if err != nil {
			log.Printf("Error looking up entity key %s: %s", kid, err.Error())
			return
		}
		mfs.keys[kid] = pubk
	}

	ok, err = verifyManifest(mf, pubk)
	switch {
	case err != nil:
		log.Printf("Error verifying manifest %s: %s", mfh, err.Error())

	case !ok:
		log.Printf("Error verifying manifest %s: signature verification failed", mfh)

	default:
		// yay! a valid manifest.
		mfs.mf[mfh] = ManifestRecord{mf, src}
	}
}

func (mfs *ManifestStoreImpl) Remove(src p2p_peer.ID) {
	mfs.mx.Lock()
	defer mfs.mx.Unlock()

	for mfh, mfr := range mfs.mf {
		if mfr.src == src {
			delete(mfs.mf, mfh)
		}
	}
}

func (mfs *ManifestStoreImpl) Lookup(entity string) []*pb.Manifest {
	mfs.mx.Lock()
	defer mfs.mx.Unlock()

	switch {
	case entity == "":
		fallthrough
	case entity == "*":
		return mfs.lookupManifest(func(*pb.Manifest) bool {
			return true
		})

	default:
		return mfs.lookupManifest(func(mf *pb.Manifest) bool {
			return mf.Entity == entity
		})
	}
}

func (mfs *ManifestStoreImpl) lookupManifest(filter func(*pb.Manifest) bool) []*pb.Manifest {
	res := make([]*pb.Manifest, 0)
	for _, mfr := range mfs.mf {
		if filter(mfr.mf) {
			res = append(res, mfr.mf)
		}
	}
	return res
}

func hashManifest(mf *pb.Manifest) (multihash.Multihash, error) {
	bytes, err := ggproto.Marshal(mf)
	if err != nil {
		return nil, err
	}

	return mc.Hash(bytes), nil
}

func verifyManifest(mf *pb.Manifest, pubk p2p_crypto.PubKey) (bool, error) {
	sig := mf.Signature
	mf.Signature = nil
	bytes, err := ggproto.Marshal(mf)
	mf.Signature = sig

	if err != nil {
		return false, err
	}

	return pubk.Verify(bytes, sig)
}

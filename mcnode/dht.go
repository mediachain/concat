package main

import (
	"context"
	ipfs_ds "github.com/ipfs/go-datastore"
	ipfs_dsq "github.com/ipfs/go-datastore/query"
	p2p_host "github.com/libp2p/go-libp2p-host"
	p2p_dht "github.com/libp2p/go-libp2p-kad-dht"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	p2p_pstore "github.com/libp2p/go-libp2p-peerstore"
	"sync"
)

type DHTImpl struct {
	dht *p2p_dht.IpfsDHT
	ds  ipfs_ds.Batching
}

func NewDHT(ctx context.Context, host p2p_host.Host) DHT {
	ds := ipfs_ds.NewLogDatastore(&IPFSDatastore{tab: ipfs_ds.NewMapDatastore()}, "DHT")
	dht := p2p_dht.NewDHTClient(ctx, host, ds)
	return &DHTImpl{dht, ds}
}

func (dht *DHTImpl) Bootstrap() error {
	// XXX Implement me
	return nil
}

func (dht *DHTImpl) Lookup(ctx context.Context, pid p2p_peer.ID) (empty p2p_pstore.PeerInfo, err error) {
	// XXX Implement me
	return empty, UnknownPeer
}

func (dht *DHTImpl) Close() error {
	return dht.dht.Close()
}

// basic ipfs_ds.MapDatastore is not thread-safe, and I couldn't find a thread-safe impl
type IPFSDatastore struct {
	tab *ipfs_ds.MapDatastore
	mx  sync.Mutex
}

func (ds *IPFSDatastore) Put(key ipfs_ds.Key, value interface{}) error {
	ds.mx.Lock()
	defer ds.mx.Unlock()
	return ds.tab.Put(key, value)
}

func (ds *IPFSDatastore) Get(key ipfs_ds.Key) (value interface{}, err error) {
	ds.mx.Lock()
	defer ds.mx.Unlock()
	return ds.tab.Get(key)
}

func (ds *IPFSDatastore) Has(key ipfs_ds.Key) (bool, error) {
	ds.mx.Lock()
	defer ds.mx.Unlock()
	return ds.tab.Has(key)
}

func (ds *IPFSDatastore) Delete(key ipfs_ds.Key) error {
	ds.mx.Lock()
	defer ds.mx.Unlock()
	return ds.tab.Delete(key)
}

// these two are not used by DHT code as far as I can tell
func (ds *IPFSDatastore) Query(q ipfs_dsq.Query) (ipfs_dsq.Results, error) {
	ds.mx.Lock()
	defer ds.mx.Unlock()
	return ds.tab.Query(q)
}

func (ds *IPFSDatastore) Batch() (ipfs_ds.Batch, error) {
	return ipfs_ds.NewBasicBatch(ds), nil
}

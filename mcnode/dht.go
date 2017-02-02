package main

import (
	"context"
	ipfs_cid "github.com/ipfs/go-cid"
	ipfs_ds "github.com/ipfs/go-datastore"
	ipfs_dsq "github.com/ipfs/go-datastore/query"
	goproc "github.com/jbenet/goprocess"
	goproc_ctx "github.com/jbenet/goprocess/context"
	goproc_per "github.com/jbenet/goprocess/periodic"
	p2p_host "github.com/libp2p/go-libp2p-host"
	p2p_dht "github.com/libp2p/go-libp2p-kad-dht"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	p2p_pstore "github.com/libp2p/go-libp2p-peerstore"
	p2p_rt "github.com/libp2p/go-libp2p-routing"
	mc "github.com/mediachain/concat/mc"
	"log"
	"math/rand"
	"sync"
	"time"
)

type DHTImpl struct {
	host  p2p_host.Host
	dht   *p2p_dht.IpfsDHT
	ds    ipfs_ds.Batching
	bootp goproc.Process
}

const (
	DHTMinPeers          = 4
	DHTMinBootstrapPeers = 2
	DHTTickPeriod        = 1 * time.Minute
	LotsOfProviders      = 2 << 20
)

func NewDHT(ctx context.Context, host p2p_host.Host) DHT {
	ds := &IPFSDatastore{tab: ipfs_ds.NewMapDatastore()}
	dht := p2p_dht.NewDHTClient(ctx, host, ds)
	return &DHTImpl{host: host, dht: dht, ds: ds}
}

func (dht *DHTImpl) Bootstrap() error {
	bootp := dht.bootstrapOverlay()
	ctx := goproc_ctx.OnClosingContext(bootp)
	// XXX This leaks a ticker on shutdown
	//     BUG: https://github.com/libp2p/go-libp2p-kad-dht/issues/43
	err := dht.dht.Bootstrap(ctx)
	if err != nil {
		bootp.Close()
		return err
	}

	dht.bootp = bootp
	return nil
}

func (dht *DHTImpl) bootstrapOverlay() goproc.Process {
	bootp := goproc_per.Tick(DHTTickPeriod, dht.bootstrapTick)
	bootp.Go(dht.bootstrapTick) // run one right now
	return bootp
}

func (dht *DHTImpl) bootstrapTick(worker goproc.Process) {
	ctx := goproc_ctx.OnClosingContext(worker)
	peers := dht.host.Network().Peers()
	switch {
	case len(peers) < DHTMinPeers:
		dht.bootstrapConnect(ctx, DHTMinPeers)

	case countBootstrapPeers(peers) < DHTMinBootstrapPeers:
		// until we are a full DHT node (with dht protocol implementation)
		// we keep a connection to at least some bootstrap peers at all times
		dht.bootstrapConnect(ctx, DHTMinBootstrapPeers)
	}
}

func (dht *DHTImpl) bootstrapConnect(ctx context.Context, count int) {
	peers := randomBootstrapPeers(count)
	for _, peer := range peers {
		log.Printf("DHT bootstrap: connecting to %s", peer.ID.Pretty())
		err := dht.host.Connect(ctx, peer)
		if err != nil {
			log.Printf("DHT bootstrap: error connecting to %s", peer.ID.Pretty())
		}
	}
}

func (dht *DHTImpl) Lookup(ctx context.Context, pid p2p_peer.ID) (pinfo p2p_pstore.PeerInfo, err error) {
	pinfo, err = dht.dht.FindPeer(ctx, pid)
	switch err {
	case nil:
		// filter unroutable addrs (eg localhost)
		pinfo.Addrs = mc.FilterAddrs(pinfo.Addrs, mc.IsRoutableAddr)

	case p2p_rt.ErrNotFound:
		err = UnknownPeer
	}
	return
}

func (dht *DHTImpl) Provide(ctx context.Context, key string) error {
	cid := ipfs_cid.NewCidV1(ipfs_cid.Raw, mc.Hash([]byte(key)))
	return dht.dht.Provide(ctx, cid)
}

func (dht *DHTImpl) FindProviders(ctx context.Context, key string) <-chan p2p_pstore.PeerInfo {
	cid := ipfs_cid.NewCidV1(ipfs_cid.Raw, mc.Hash([]byte(key)))
	return dht.dht.FindProvidersAsync(ctx, cid, LotsOfProviders)
}

func (dht *DHTImpl) Close() error {
	if dht.bootp != nil {
		err := dht.bootp.Close()
		if err != nil {
			log.Printf("Error closing DHT bootstrap process: %s", err.Error())
		}
	}

	return dht.dht.Close()
}

// Bootstrap peers; we use the mainline IPFS bootstrap peers
var DHTBootstrapPeers = []string{
	"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",  // mars.i.ipfs.io
	"/ip4/104.236.176.52/tcp/4001/p2p/QmSoLnSGccFuZQJzRadHn95W2CrSFmZuTdDWP8HXaHca9z",  // neptune.i.ipfs.io
	"/ip4/104.236.179.241/tcp/4001/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM", // pluto.i.ipfs.io
	"/ip4/162.243.248.213/tcp/4001/p2p/QmSoLueR4xBeUbY9WZ9xGUUxunbKWcrNFTDAadQJmocnWm", // uranus.i.ipfs.io
	"/ip4/128.199.219.111/tcp/4001/p2p/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu", // saturn.i.ipfs.io
	"/ip4/104.236.76.40/tcp/4001/p2p/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",   // venus.i.ipfs.io
	"/ip4/178.62.158.247/tcp/4001/p2p/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",  // earth.i.ipfs.io
	"/ip4/178.62.61.185/tcp/4001/p2p/QmSoLMeWqB7YGVLJN3pNLQpmmEk35v6wYtsMGLzSr5QBU3",   // mercury.i.ipfs.io
	"/ip4/104.236.151.122/tcp/4001/p2p/QmSoLju6m7xTh3DuokvT3886QRYqxAzb1kShaanJgW36yx", // jupiter.i.ipfs.io
}

var (
	DHTBootstrapPeerInfo []p2p_pstore.PeerInfo
	DHTBootstrapPeerSet  map[p2p_peer.ID]bool
)

func init() {
	DHTBootstrapPeerInfo = make([]p2p_pstore.PeerInfo, len(DHTBootstrapPeers))
	DHTBootstrapPeerSet = make(map[p2p_peer.ID]bool)
	for x, peer := range DHTBootstrapPeers {
		pinfo, err := mc.ParseHandle(peer)
		if err != nil {
			log.Fatal(err)
		}
		DHTBootstrapPeerInfo[x] = pinfo
		DHTBootstrapPeerSet[pinfo.ID] = true
	}
}

func countBootstrapPeers(peers []p2p_peer.ID) int {
	count := 0
	for _, pid := range peers {
		if DHTBootstrapPeerSet[pid] {
			count += 1
		}
	}
	return count
}

func randomBootstrapPeers(count int) []p2p_pstore.PeerInfo {
	peerCount := len(DHTBootstrapPeerInfo)
	if count >= peerCount {
		return DHTBootstrapPeerInfo
	}

	peers := make([]p2p_pstore.PeerInfo, count)
	for x, y := range rand.Perm(peerCount)[:count] {
		peers[x] = DHTBootstrapPeerInfo[y]
	}

	return peers
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

func (ds *IPFSDatastore) Query(q ipfs_dsq.Query) (ipfs_dsq.Results, error) {
	ds.mx.Lock()
	defer ds.mx.Unlock()
	return ds.tab.Query(q)
}

func (ds *IPFSDatastore) Batch() (ipfs_ds.Batch, error) {
	return ipfs_ds.NewBasicBatch(ds), nil
}

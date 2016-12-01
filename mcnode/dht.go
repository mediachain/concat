package main

import (
	"context"
	p2p_host "github.com/libp2p/go-libp2p-host"
	p2p_dht "github.com/libp2p/go-libp2p-kad-dht"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	p2p_pstore "github.com/libp2p/go-libp2p-peerstore"
)

type DHTImpl struct {
	dht *p2p_dht.IpfsDHT
}

func NewDHT(ctx context.Context, host p2p_host.Host) (DHT, error) {
	// XXX Implement me
	return &DHTImpl{}, nil
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
	// XXX Implement me
	return nil
}

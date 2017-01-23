package main

import (
	"context"
	"flag"
	"fmt"
	p2p_peer "github.com/libp2p/go-libp2p-peer"
	mc "github.com/mediachain/concat/mc"
	homedir "github.com/mitchellh/go-homedir"
	"log"
	"os"
)

func main() {
	mc.SetLibp2pClient("mcdir")

	port := flag.Int("l", 9000, "Listen port")
	hdir := flag.String("d", "~/.mediachain/mcdir", "Directory home")
	ver := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if len(flag.Args()) != 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options ...]\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *ver {
		fmt.Println(mc.ConcatVersion)
		os.Exit(0)
	}

	home, err := homedir.Expand(*hdir)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(home, 0755)
	if err != nil {
		log.Fatal(err)
	}

	id, err := mc.MakePeerIdentity(home)
	if err != nil {
		log.Fatal(err)
	}

	addr, err := mc.ParseAddress(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port))
	if err != nil {
		log.Fatal(err)
	}

	host, err := mc.NewHost(context.Background(), id, addr)
	if err != nil {
		log.Fatal(err)
	}

	dir := &Directory{PeerIdentity: id, host: host, peers: make(map[p2p_peer.ID]PeerRecord)}
	host.SetStreamHandler("/mediachain/dir/register", dir.registerHandler)
	host.SetStreamHandler("/mediachain/dir/lookup", dir.lookupHandler)
	host.SetStreamHandler("/mediachain/dir/list", dir.listHandler)
	host.SetStreamHandler("/mediachain/dir/listns", dir.listnsHandler)

	for _, addr := range host.Addrs() {
		if !mc.IsLinkLocalAddr(addr) {
			log.Printf("I am %s/p2p/%s", addr, id.Pretty())
		}
	}
	select {}
}

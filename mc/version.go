package mc

import (
	"fmt"
	p2p_id "github.com/libp2p/go-libp2p/p2p/protocol/identify"
)

const ConcatVersion = "1.6"
const Libp2pVersion = "go-libp2p/4.3.1"

func SetLibp2pClient(prog string) {
	p2p_id.ClientVersion = fmt.Sprintf("%s/%s (%s)", prog, ConcatVersion, Libp2pVersion)
}

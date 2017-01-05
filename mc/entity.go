package mc

import (
	"errors"
	p2p_crypto "github.com/libp2p/go-libp2p-crypto"
)

func LookupEntityKey(entity string, keyId string) (p2p_crypto.PubKey, error) {
	return nil, errors.New("IMPLEMENT ME: mc.LookupEntityKey")
}

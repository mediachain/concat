package mc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	p2p_crypto "github.com/libp2p/go-libp2p-crypto"
	multihash "github.com/multiformats/go-multihash"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
)

var (
	MalformedEntityId = errors.New("Malformed entity id")
	UnknownIdProvider = errors.New("Unknwon identity provider")
	EntityKeyNotFound = errors.New("Entity key not found")
)

type EntityId struct {
	KeyId string `json:"keyId"` // public key multihash
	Key   []byte `json:"key"`   // marshalled public key
}

func LookupEntityKey(entity string, keyId string) (p2p_crypto.PubKey, error) {
	ix := strings.Index(entity, ":")
	if ix < 0 {
		return nil, MalformedEntityId
	}

	prov := entity[:ix]
	user := entity[ix+1:]

	lookup, ok := idProviders[prov]
	if ok {
		return lookup(user, keyId)
	}

	return nil, UnknownIdProvider
}

type LookupKeyFunc func(user, keyId string) (p2p_crypto.PubKey, error)

var bsrx *regexp.Regexp

func init() {
	rx, err := regexp.Compile("^[a-zA-Z0-9.-]+$")
	if err != nil {
		log.Fatal(err)
	}
	bsrx = rx
}

func lookupBlockstack(user, keyId string) (p2p_crypto.PubKey, error) {
	if !bsrx.Match([]byte(user)) {
		return nil, MalformedEntityId
	}

	khash, err := multihash.FromB58String(keyId)
	if err != nil {
		return nil, err
	}

	out, err := exec.Command("blockstack", "lookup", user).Output()
	if err != nil {
		xerr, ok := err.(*exec.ExitError)
		if ok {
			return nil, fmt.Errorf("blockstack error: %s", string(xerr.Stderr))
		}
		return nil, err
	}

	var res map[string]interface{}
	err = json.Unmarshal(out, &res)
	if err != nil {
		return nil, err
	}

	prof, ok := res["profile"].(map[string]interface{})
	if !ok {
		return nil, EntityKeyNotFound
	}

	accts, ok := prof["account"].([]interface{})
	if !ok {
		return nil, EntityKeyNotFound
	}

	for _, acct := range accts {
		xacct, ok := acct.(map[string]interface{})
		if !ok {
			continue
		}

		svc, ok := xacct["service"].(string)
		if !ok {
			continue
		}

		if svc != "mediachain" {
			continue
		}

		key, ok := xacct["identifier"].(string)
		if !ok {
			break
		}

		return unmarshalEntityKey(key, khash)
	}

	return nil, EntityKeyNotFound
}

func lookupKeybase(user, keyId string) (p2p_crypto.PubKey, error) {
	if !bsrx.Match([]byte(user)) {
		return nil, MalformedEntityId
	}

	khash, err := multihash.FromB58String(keyId)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://%s.keybase.pub/mediachain.json", user)

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch {
	case res.StatusCode == 404:
		return nil, EntityKeyNotFound

	case res.StatusCode != 200:
		return nil, fmt.Errorf("keybase error: %d %s", res.StatusCode, res.Status)
	}

	var pub EntityId
	err = json.NewDecoder(res.Body).Decode(&pub)
	if err != nil {
		return nil, err
	}

	if pub.KeyId != keyId {
		return nil, EntityKeyNotFound
	}

	return unmarshalEntityKeyBytes(pub.Key, khash)
}

var idProviders = map[string]LookupKeyFunc{
	"blockstack": lookupBlockstack,
	"keybase":    lookupKeybase,
}

func unmarshalEntityKey(key string, khash multihash.Multihash) (p2p_crypto.PubKey, error) {
	data, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}

	return unmarshalEntityKeyBytes(data, khash)
}

func unmarshalEntityKeyBytes(key []byte, khash multihash.Multihash) (p2p_crypto.PubKey, error) {
	hash, err := multihash.Sum(key, int(khash[0]), -1)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(hash, khash) {
		return nil, EntityKeyNotFound
	}

	return p2p_crypto.UnmarshalPublicKey(key)
}

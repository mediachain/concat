package main

import (
	"encoding/json"
	"errors"
	ggproto "github.com/gogo/protobuf/proto"
	//b58 "github.com/jbenet/go-base58"
	p2p_crypto "github.com/libp2p/go-libp2p-crypto"
	//mc "github.com/mediachain/concat/mc"
	pb "github.com/mediachain/concat/proto"
	homedir "github.com/mitchellh/go-homedir"
	kp "gopkg.in/alecthomas/kingpin.v2"
	"log"
	"os"
	"time"
)

func main() {
	log.SetFlags(0) // naked logs, it's interactive error output

	var (
		home = kp.Flag("home", "mcid home directory").Short('d').Default("~/.mediachain/mcid").String()

		_ = kp.Command("id", "show your public identity; generates a new key pair if it doesn't already exist.") // idCmd declared but not used

		signCmd      = kp.Command("sign", "sign a manifest")
		signEntity   = signCmd.Arg("entity", "entity id").Required().String()
		signManifest = signCmd.Arg("manifest", "manifest json file").Required().File()

		verifyCmd      = kp.Command("verify", "verify a manifest")
		verifyManifest = verifyCmd.Arg("manifest", "manifest json file").Required().ExistingFile()
	)

	switch kp.Parse() {
	case "id":
		doId(*home)

	case "sign":
		doSign(*home, *signEntity, *signManifest)

	case "verify":
		doVerify(*home, *verifyManifest)
	}
}

type Identity struct {
	Public  PublicId  `json:"public"`
	Private PrivateId `json:"private"`
}

type PublicId struct {
	KeyId string `json:"keyId"` // public key multihash
	Key   []byte `json:"key"`   // marshalled public key
}

type PrivateId struct {
	Params ScryptParams `json:"params"` // key derivation parameters
	Salt   []byte       `json:"salt"`   // key derivation salt
	Nonce  []byte       `json:"nonce"`  // encryption nonce
	Data   []byte       `json:"data"`   // encrypted marshalled private key
}

type ScryptParams struct {
	N, R, P int
}

// ops
func doId(home string) {
	id, err := getIdentity(home, true) // generate id if it doesn't already exist
	if err != nil {
		log.Fatalf("Error retrieving identity: %s", err.Error())
	}

	json.NewEncoder(os.Stdout).Encode(id.Public)
}

func doSign(home string, entity string, mf *os.File) {
	var manifest pb.Manifest
	var manifestBody pb.ManifestBody

	err := json.NewDecoder(mf).Decode(&manifestBody)
	if err != nil {
		log.Fatalf("Error decoding manifest body: %s", err.Error())
	}

	id, err := getIdentity(home, false) // error if id doesn't exist
	if err != nil {
		log.Fatalf("Error retrieving identity: %s", err.Error())
	}

	privk, err := getPrivateKey(id.Private)
	if err != nil {
		log.Fatalf("Error decrypting private key: %s", err.Error())
	}

	manifest.Entity = entity
	manifest.KeyId = id.Public.KeyId
	manifest.Body = &manifestBody
	manifest.Timestamp = time.Now().Unix()

	bytes, err := ggproto.Marshal(&manifest)
	if err != nil {
		log.Fatalf("Error marshalling manifest: %s", err.Error())
	}

	sig, err := privk.Sign(bytes)
	if err != nil {
		log.Fatalf("Error signing manifest: %s", err.Error())
	}

	manifest.Signature = sig

	json.NewEncoder(os.Stdout).Encode(&manifest)
}

func doVerify(home string, manifest string) {
	log.Fatalf("IMPLEMENT ME: doVerify %s %s\n", home, manifest)
}

// identity
func getIdentity(home string, gen bool) (id Identity, err error) {
	home, err = homedir.Expand(home)
	if err != nil {
		return
	}

	return id, errors.New("IMPLEMENT ME: getIdentity")
}

func getPrivateKey(priv PrivateId) (p2p_crypto.PrivKey, error) {
	bytes, err := decryptPrivateId(priv)
	if err != nil {
		return nil, err
	}

	return p2p_crypto.UnmarshalPrivateKey(bytes)
}

func decryptPrivateId(priv PrivateId) ([]byte, error) {
	return nil, errors.New("IMPLEMENT ME: decryptPrivateId")
}

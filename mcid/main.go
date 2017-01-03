package main

import (
	"encoding/json"
	"errors"
	"fmt"
	b58 "github.com/jbenet/go-base58"
	p2p_crypto "github.com/libp2p/go-libp2p-crypto"
	mc "github.com/mediachain/concat/mc"
	kp "gopkg.in/alecthomas/kingpin.v2"
	"log"
	"os"
)

func main() {
	var (
		home = kp.Flag("home", "mcid home directory").Short('d').Default("~/.mediachain/mcid").String()

		_ = kp.Command("id", "show your public identity; generates a new key pair if it doesn't already exist.") // idCmd declared but not used

		signCmd      = kp.Command("sign", "sign a manifest")
		signEntity   = signCmd.Arg("entity", "entity id").Required().String()
		signManifest = signCmd.Arg("manifest", "manifest json file").Required().ExistingFile()

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

type PublicId struct {
	KeyId string `json:"keyId"`
	Key   []byte `json:"key"`
}

func doId(home string) {
	pubk, err := getPublicKey(home)
	if err != nil {
		log.Fatalf("Error retrieving public key: %s", err.Error())
	}

	bytes, err := pubk.Bytes()
	if err != nil {
		log.Fatalf("Error marshalling public key: %s", err.Error())
	}

	id := b58.Encode(mc.Hash(bytes))

	json.NewEncoder(os.Stdout).Encode(PublicId{id, bytes})
}

func doSign(home string, entity string, manifest string) {
	fmt.Printf("IMPLEMENT ME: doSign %s %s %s\n", home, entity, manifest)
}

func doVerify(home string, manifest string) {
	fmt.Printf("IMPLEMENT ME: doVerify %s %s\n", home, manifest)
}

func getPublicKey(home string) (p2p_crypto.PubKey, error) {
	return nil, errors.New("IMPLEMENT ME: getPublicKey")
}

func getPrivateKey(home string) (p2p_crypto.PrivKey, error) {
	return nil, errors.New("IMPLEMENT ME: getPrivateKey")
}

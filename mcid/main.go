package main

import (
	"encoding/json"
	"errors"
	ggproto "github.com/gogo/protobuf/proto"
	b58 "github.com/jbenet/go-base58"
	p2p_crypto "github.com/libp2p/go-libp2p-crypto"
	mc "github.com/mediachain/concat/mc"
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

func doSign(home string, entity string, mf *os.File) {
	var manifest pb.Manifest
	var manifestBody pb.ManifestBody

	err := json.NewDecoder(mf).Decode(&manifestBody)
	if err != nil {
		log.Fatalf("Error decoding manifest body: %s", err.Error())
	}

	privk, err := getPrivateKey(home)
	if err != nil {
		log.Fatalf("Error retrieving private key: %s", err.Error())
	}

	pubk := privk.GetPublic()
	pbytes, err := pubk.Bytes()
	if err != nil {
		log.Fatalf("Error marshalling public key: %s", err.Error())
	}
	id := b58.Encode(mc.Hash(pbytes))

	manifest.Entity = entity
	manifest.KeyId = id
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

func getPublicKey(home string) (p2p_crypto.PubKey, error) {
	_, err := homedir.Expand(home)
	if err != nil {
		return nil, err
	}

	return nil, errors.New("IMPLEMENT ME: getPublicKey")
}

func getPrivateKey(home string) (p2p_crypto.PrivKey, error) {
	_, err := homedir.Expand(home)
	if err != nil {
		return nil, err
	}

	return nil, errors.New("IMPLEMENT ME: getPrivateKey")
}

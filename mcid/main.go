package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	jsonpb "github.com/gogo/protobuf/jsonpb"
	ggproto "github.com/gogo/protobuf/proto"
	b58 "github.com/jbenet/go-base58"
	p2p_crypto "github.com/libp2p/go-libp2p-crypto"
	mc "github.com/mediachain/concat/mc"
	pb "github.com/mediachain/concat/proto"
	gopass "github.com/mediachain/gopass"
	homedir "github.com/mitchellh/go-homedir"
	sbox "golang.org/x/crypto/nacl/secretbox"
	scrypt "golang.org/x/crypto/scrypt"
	kp "gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"
)

func main() {
	log.SetFlags(0) // naked logs, it's interactive output

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

// default scrypt parameters (2009): N=16384, r=8, p=1
// still current; see https://github.com/Tarsnap/scrypt/issues/19
const (
	ScryptN = 16384
	ScryptR = 8
	ScryptP = 1
)

var (
	DecryptionError = errors.New("Private key decryption failed")
)

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

	err := jsonpb.Unmarshal(mf, &manifestBody)
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

	marshaler := jsonpb.Marshaler{}
	err = marshaler.Marshal(os.Stdout, &manifest)
	if err != nil {
		log.Fatalf("Error encoding manifest: %s", err.Error())
	}
	fmt.Println()
}

func doVerify(home string, manifest string) {
	log.Fatalf("IMPLEMENT ME: doVerify %s %s\n", home, manifest)
}

// identity
func getIdentity(home string, genid bool) (id Identity, err error) {
	home, err = homedir.Expand(home)
	if err != nil {
		return
	}

	idpath := path.Join(home, "identity.json")
	_, err = os.Stat(idpath)
	switch {
	case os.IsNotExist(err):
		if genid {
			return generateIdentity(home, idpath)
		}
		fallthrough
	case err != nil:
		return

	default:
		return loadIdentity(idpath)
	}
}

func generateIdentity(home, idpath string) (id Identity, err error) {
	err = os.MkdirAll(home, 0755)
	if err != nil {
		return
	}

	log.Printf("Generating identity key pair")
	privk, pubk, err := mc.GenerateECCKeyPair()
	if err != nil {
		return
	}

	pubbytes, err := pubk.Bytes()
	if err != nil {
		return
	}
	kid := b58.Encode(mc.Hash(pubbytes))

	id.Public.KeyId = kid
	id.Public.Key = pubbytes

	privbytes, err := privk.Bytes()
	if err != nil {
		return
	}

	err = encryptPrivateId(&id.Private, privbytes)
	if err != nil {
		return
	}

	bytes, err := json.Marshal(&id)
	if err != nil {
		return
	}

	err = ioutil.WriteFile(idpath, bytes, 0600)
	return
}

func loadIdentity(idpath string) (id Identity, err error) {
	bytes, err := ioutil.ReadFile(idpath)
	if err != nil {
		return
	}

	err = json.Unmarshal(bytes, &id)
	return
}

func getPrivateKey(priv PrivateId) (p2p_crypto.PrivKey, error) {
	bytes, err := decryptPrivateId(priv)
	if err != nil {
		return nil, err
	}

	return p2p_crypto.UnmarshalPrivateKey(bytes)
}

// private key encryption/decryption:
//  key derivation with scrypt
//  encryption with nacl secretbox
func encryptPrivateId(priv *PrivateId, data []byte) error {
	var (
		salt  [16]byte
		nonce [24]byte
		key   [32]byte
	)

	_, err := rand.Read(salt[:])
	if err != nil {
		return err
	}

	_, err = rand.Read(nonce[:])
	if err != nil {
		return err
	}

	pass, err := getEncryptionPass()
	if err != nil {
		return err
	}

	xkey, err := scrypt.Key(pass, salt[:], ScryptN, ScryptR, ScryptP, 32)
	if err != nil {
		return err
	}

	copy(key[:], xkey)

	ctext := sbox.Seal(nil, data, &nonce, &key)

	priv.Salt = salt[:]
	priv.Nonce = nonce[:]
	priv.Data = ctext
	priv.Params.N = ScryptN
	priv.Params.R = ScryptR
	priv.Params.P = ScryptP

	return nil
}

func decryptPrivateId(priv PrivateId) ([]byte, error) {
	var (
		nonce [24]byte
		key   [32]byte
	)

	pass, err := getDecryptionPass()
	if err != nil {
		return nil, err
	}

	xkey, err := scrypt.Key(pass, priv.Salt, priv.Params.N, priv.Params.R, priv.Params.P, 32)
	if err != nil {
		return nil, err
	}

	copy(nonce[:], priv.Nonce)
	copy(key[:], xkey)

	bytes, ok := sbox.Open(nil, priv.Data, &nonce, &key)
	if !ok {
		return nil, DecryptionError
	}

	return bytes, nil
}

func getEncryptionPass() ([]byte, error) {
	for {
		fmt.Fprintf(os.Stderr, "Enter passphrase: ")
		pass1, err := gopass.GetPasswdX(false, os.Stderr)
		if err != nil {
			return nil, err
		}

		fmt.Fprintf(os.Stderr, "Re-enter passphrase: ")
		pass2, err := gopass.GetPasswdX(false, os.Stderr)
		if err != nil {
			return nil, err
		}

		if bytes.Equal(pass1, pass2) {
			return pass1, nil
		}

		fmt.Fprintln(os.Stderr, "Passphrases don't match")
	}
}

func getDecryptionPass() ([]byte, error) {
	fmt.Fprintf(os.Stderr, "Enter passphrase: ")
	return gopass.GetPasswdX(false, os.Stderr)
}

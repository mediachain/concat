package main

import (
	"fmt"
	kp "gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	var (
		home = kp.Flag("home", "mcid home directory").Short('d').Default("~/.mediachain/mcid").String()

		_ = kp.Command("id", "show your public identity; generates a new key pair if it doesn't exist") // idCmd declared but not used

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

func doId(home string) {
	fmt.Printf("IMPLEMENT ME: doId %s\n", home)
}

func doSign(home string, entity string, manifest string) {
	fmt.Printf("IMPLEMENT ME: doSign %s %s %s\n", home, entity, manifest)
}

func doVerify(home string, manifest string) {
	fmt.Printf("IMPLEMENT ME: doVerify %s %s\n", home, manifest)
}

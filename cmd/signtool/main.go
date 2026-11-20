// Command signtool signs release checksums so the self-updater can verify
// that a release really came from this project.
//
//	go run ./cmd/signtool genkey          # prints a new private and public key
//	UPDATE_SIGNING_KEY=... go run ./cmd/signtool sign checksums.txt
//	go run ./cmd/signtool verify checksums.txt <public-key>
//
// The private key is the base64 ed25519 seed (store it as the
// UPDATE_SIGNING_KEY GitHub secret); the public key is embedded in
// pkg/update. Signatures are written to <file>.sig as base64.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "genkey":
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		check(err)
		fmt.Println("private (UPDATE_SIGNING_KEY secret):", base64.StdEncoding.EncodeToString(priv.Seed()))
		fmt.Println("public  (pkg/update PublicKey):    ", base64.StdEncoding.EncodeToString(pub))
	case "sign":
		if len(os.Args) != 3 {
			usage()
		}
		seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(os.Getenv("UPDATE_SIGNING_KEY")))
		if err != nil || len(seed) != ed25519.SeedSize {
			fail("UPDATE_SIGNING_KEY must be a base64 ed25519 seed")
		}
		data, err := os.ReadFile(os.Args[2])
		check(err)
		sig := ed25519.Sign(ed25519.NewKeyFromSeed(seed), data)
		check(os.WriteFile(os.Args[2]+".sig", []byte(base64.StdEncoding.EncodeToString(sig)+"\n"), 0o644))
		fmt.Println("signed", os.Args[2])
	case "verify":
		if len(os.Args) != 4 {
			usage()
		}
		pub, err := base64.StdEncoding.DecodeString(os.Args[3])
		check(err)
		data, err := os.ReadFile(os.Args[2])
		check(err)
		sigB64, err := os.ReadFile(os.Args[2] + ".sig")
		check(err)
		sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigB64)))
		check(err)
		if len(pub) != ed25519.PublicKeySize || !ed25519.Verify(pub, data, sig) {
			fail("signature INVALID")
		}
		fmt.Println("signature OK")
	default:
		usage()
	}
}

func usage() { fail("usage: signtool genkey | sign <file> | verify <file> <public-key>") }

func check(err error) {
	if err != nil {
		fail(err.Error())
	}
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

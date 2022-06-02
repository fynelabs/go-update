package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/urfave/cli/v2"
)

func Check() *cli.Command {
	a := &Application{}

	return &cli.Command{
		Name:        "check",
		Usage:       "Check that a signature for a Fyne binary in FyneApp.toml is correct",
		Description: "You may specify a filename for the Private Key, the executable and the FyneApp.toml you want to check",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "public-key",
				Aliases:     []string{"pub"},
				Usage:       "The public key file to use to verify the signature for this executable.",
				Destination: &a.PublicKey,
				Value:       "ed25519.pem",
			},
			&cli.StringFlag{
				Name:        "executable",
				Aliases:     []string{"exe"},
				Usage:       "The executable to check the signature for.",
				Destination: &a.Executable,
			},
		},
		Action: func(ctx *cli.Context) error {
			if a.Executable != "" {
				err := a.Check(a.Executable)
				if err != nil {
					return err
				}
			}

			for _, exe := range ctx.Args().Slice() {
				err := a.Check(exe)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func (a *Application) Check(executable string) error {
	verifier, err := publicKeyVerifier(a.PublicKey)
	if err != nil {
		return err
	}

	content, err := executableContent(executable)
	if err != nil {
		return err
	}

	byteSignature, err := readSignature(executable)
	if err != nil {
		return err
	}

	ok := ed25519.Verify(verifier, content, byteSignature[:])
	if !ok {
		return fmt.Errorf("unable to verify signature")
	}
	return nil
}

func publicKeyVerifier(publicKey string) (ed25519.PublicKey, error) {
	publicKeyFile, err := os.Open(publicKey)
	if err != nil {
		return []byte{}, err
	}
	defer publicKeyFile.Close()

	b, err := ioutil.ReadAll(publicKeyFile)
	if err != nil {
		return []byte{}, err
	}

	block, _ := pem.Decode(b)
	if block == nil {
		return []byte{}, fmt.Errorf("unable to decode Public Key PEM")
	}

	signer, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return []byte{}, nil
	}

	ed25519verifier, ok := signer.(ed25519.PublicKey)
	if !ok {
		return []byte{}, fmt.Errorf("public Key is not an ED25519")
	}

	return ed25519verifier, nil
}

func readSignature(executable string) ([64]byte, error) {
	signatureFile, err := os.Open(executable + ".ed25519")
	if err != nil {
		return [64]byte{}, err
	}

	info, err := signatureFile.Stat()
	if err != nil {
		return [64]byte{}, err
	}

	if info.Size() != 64 {
		return [64]byte{}, fmt.Errorf("ed25519 signature must be 64 bytes long and was %v", info.Size())
	}

	writer := bytes.NewBuffer(make([]byte, 0, 64))
	n, err := io.Copy(writer, signatureFile)
	if err != nil {
		return [64]byte{}, err
	}

	if n != 64 {
		return [64]byte{}, fmt.Errorf("ed25519 signature must be 64 bytes long and was %v", n)
	}

	return *(*[64]byte)(writer.Bytes()), nil
}

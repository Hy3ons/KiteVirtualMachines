package sshkey

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"

	"golang.org/x/crypto/ssh"
)

type KeyPair struct {
	PrivateKey string
	PublicKey  string
}

// GenerateRSA creates one SSH key pair for a KiteVirtualMachine.
// bits controls the RSA key size; use 2048 or larger for VM login keys.
// The returned KeyPair stores the private key as PEM and public key in authorized_keys format.
func GenerateRSA(bits int) (KeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return KeyPair{}, err
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if privateKeyPEM == nil {
		return KeyPair{}, errors.New("failed to encode private key")
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return KeyPair{}, err
	}

	return KeyPair{
		PrivateKey: string(privateKeyPEM),
		PublicKey:  strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey))),
	}, nil
}

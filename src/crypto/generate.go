package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
)

func GenerateKeys() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	publicKey, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, nil, errors.New("casting error ")
	}

	return privateKey, publicKey, nil
}

func GenerateSignature(data []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {

	hashed := sha256.Sum256(data)

	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hashed[:])
	if err != nil {
		return []byte{}, nil
	}

	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])

	return signature, nil
}

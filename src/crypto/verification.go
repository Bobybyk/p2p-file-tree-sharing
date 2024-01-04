package crypto

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"math/big"
)

func VerifyMessage(data []byte, signature []byte, publicKey *ecdsa.PublicKey) bool {

	var r, s big.Int
	r.SetBytes(signature[:32])
	s.SetBytes(signature[32:])
	hashed := sha256.Sum256(data)
	return ecdsa.Verify(publicKey, hashed[:], &r, &s)
}

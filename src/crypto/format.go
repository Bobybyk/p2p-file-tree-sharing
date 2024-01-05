package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"
)

func FormatPublicKey(publicKey ecdsa.PublicKey) []byte {

	formatted := make([]byte, 64)
	publicKey.X.FillBytes(formatted[:32])
	publicKey.Y.FillBytes(formatted[32:])

	return formatted
}

func ParsePublicKey(key []byte) ecdsa.PublicKey {
	var x, y big.Int
	x.SetBytes(key[:32])
	y.SetBytes(key[32:])
	publicKey := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     &x,
		Y:     &y,
	}
	return publicKey
}

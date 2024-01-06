package crypto

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/rapidloop/skv"
)

func LoadFromDisk(filepath string) (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	var privateKey *ecdsa.PrivateKey
	var publicKey *ecdsa.PublicKey

	//Private and Public key Generation or retrieval from bank
	keysFile, err := skv.Open(filepath)
	defer keysFile.Close()

	if err != nil {
		return &ecdsa.PrivateKey{}, &ecdsa.PublicKey{}, errors.New("could open file: " + err.Error())
	}

	var privateKeyString string
	var publicKeyString string

	err = keysFile.Get("private", &privateKeyString)
	if err != nil {
		fmt.Println("could not get private key: ", err.Error(), "\nGenerating new pair...")

		privateKey, publicKey, err = GenerateKeys()
		if err != nil {
			return &ecdsa.PrivateKey{}, &ecdsa.PublicKey{}, errors.New("could not generate keys: " + err.Error())
		}

		privateKeyString, publicKeyString = EncodeToString(privateKey, publicKey)

		err := keysFile.Put("private", privateKeyString)
		if err != nil {
			return &ecdsa.PrivateKey{}, &ecdsa.PublicKey{}, errors.New("could not store private key: " + err.Error())
		}

		err = keysFile.Put("public", publicKeyString)
		if err != nil {
			return &ecdsa.PrivateKey{}, &ecdsa.PublicKey{}, errors.New("could not store public key: " + err.Error())
		}
	}

	err = keysFile.Get("public", &publicKeyString)
	if err != nil {
		return &ecdsa.PrivateKey{}, &ecdsa.PublicKey{}, errors.New("could not get public key: " + err.Error())
	}

	privateKey, publicKey = DecodeFromString(privateKeyString, publicKeyString)

	return privateKey, publicKey, nil
}

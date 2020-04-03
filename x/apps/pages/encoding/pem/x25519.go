package pem

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
)

const (
	blockTypePrivateKey = "PRIVATE KEY"
	blockTypePublicKey  = "PUBLIC KEY"
	algoX25519          = "1.3.101.110"
)

// pkcs8 reflects an ASN.1, PKCS#8 PrivateKey. See
// ftp://ftp.rsasecurity.com/pub/pkcs/pkcs-8/pkcs-8v1_2.asn
// and RFC 5208.
//
// (copied from crypto/x509/pkcs8.go)
type pkcs8priv struct {
	Version    int
	Algo       pkix.AlgorithmIdentifier
	PrivateKey []byte
	// optional attributes omitted.
}

type pkcs8pub struct {
	Algo      pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

func ParseX25519PrivateKey(in []byte) (privateKey []byte, err error) {
	block, _ := pem.Decode(in)
	if block == nil {
		return nil, errors.New("failed to decode the PEM block")
	}

	if block.Type != blockTypePrivateKey {
		return nil, fmt.Errorf("block type is not %s", blockTypePrivateKey)
	}

	var privKey pkcs8priv
	if _, err = asn1.Unmarshal(block.Bytes, &privKey); err != nil {
		return nil, fmt.Errorf("asn1 block: %w", err)
	}

	if privKey.Algo.Algorithm.String() != algoX25519 {
		return nil, fmt.Errorf("algorithm is not %s", algoX25519)
	}

	var curvePrivateKey []byte
	if _, err = asn1.Unmarshal(privKey.PrivateKey, &curvePrivateKey); err != nil {
		return nil, fmt.Errorf("private key bytes: %w", err)
	}

	return curvePrivateKey, nil
}

func ParseX25519PublicKey(in []byte) (publicKey []byte, err error) {
	block, _ := pem.Decode(in)
	if block == nil {
		return nil, errors.New("failed to decode the PEM block")
	}

	var pubKey pkcs8pub
	if _, err = asn1.Unmarshal(block.Bytes, &pubKey); err != nil {
		return nil, fmt.Errorf("asn1 block: %w", err)
	}

	var pubKey2 interface{}
	if _, err = asn1.Unmarshal(block.Bytes, &pubKey2); err != nil {
		return nil, fmt.Errorf("asn1 block: %w", err)
	}

	if pubKey.Algo.Algorithm.String() != algoX25519 {
		return nil, fmt.Errorf("algorithm is not %s", algoX25519)
	}

	return pubKey.PublicKey.Bytes, nil
}

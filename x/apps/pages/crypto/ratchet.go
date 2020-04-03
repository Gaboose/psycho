package crypto

import (
	"crypto/rand"

	"golang.org/x/crypto/nacl/box"
)

type RatchetAction int

const (
	RatchetAdvanced RatchetAction = iota + 1
	RatchetNoop
)

// DHRatchet implements Diffie-Hellman ratchet
// (https://signal.org/docs/specifications/doubleratchet/#diffie-hellman-ratchet)
// using curve25519 and the golang.org/x/crypto/nacl/box package.
type DHRatchet struct {
	prevPeersPublicKey,
	peersPublicKey,
	privateKey,
	nextPrivateKey,
	nextPublicKey *[32]byte
}

func NewDHRatchet(peersPublicKey, privateKey *[32]byte, initiator bool) *DHRatchet {
	dhr := &DHRatchet{
		prevPeersPublicKey: &[32]byte{},
		peersPublicKey:     peersPublicKey,
		privateKey:         &[32]byte{},
		nextPrivateKey:     privateKey,
		nextPublicKey:      &[32]byte{},
	}

	if initiator {
		// advance half a ratchet: our private keys,
		// but not the peers public keys
		dhr.privateKey = dhr.nextPrivateKey

		var err error
		dhr.nextPublicKey, dhr.nextPrivateKey, err = box.GenerateKey(rand.Reader)
		if err != nil {
			panic(err)
		}
	}

	return dhr
}

func (dhr *DHRatchet) Encrypt(m []byte) Packet {
	payload := newBoxPlaintext(dhr.nextPublicKey, m)

	nonce := [24]byte{}
	rand.Read(nonce[:])

	return newPacket(&nonce, payload, dhr.peersPublicKey, dhr.privateKey)
}

func (dhr *DHRatchet) Decrypt(pkt Packet) (m []byte, ok bool, act RatchetAction) {

	if !pkt.valid() {
		return nil, false, RatchetNoop
	}

	// first message with the new dhr.peersPublicKey?
	if bp, ok := pkt.open(dhr.peersPublicKey, dhr.nextPrivateKey); ok {
		dhr.advanceRatchet(bp.ratchetPublicKey())
		return bp.payload(), true, RatchetAdvanced
	}

	// a message from the previous ratchet step?
	if dhr.prevPeersPublicKey != nil {
		if bp, ok := pkt.open(dhr.prevPeersPublicKey, dhr.privateKey); ok {
			return bp.payload(), true, RatchetNoop
		}
	}

	return nil, false, RatchetNoop
}

func (dhr *DHRatchet) advanceRatchet(ratchetPublicKey *[32]byte) {
	dhr.prevPeersPublicKey = dhr.peersPublicKey
	dhr.peersPublicKey = ratchetPublicKey

	dhr.privateKey = dhr.nextPrivateKey
	var err error
	dhr.nextPublicKey, dhr.nextPrivateKey, err = box.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
}

type Packet []byte

func (p Packet) nonce() *[24]byte {
	ret := [24]byte{}
	copy(ret[:], p[:24])
	return &ret
}

func newPacket(
	nonce *[24]byte,
	plaintext []byte,
	peersPublicKey, privateKey *[32]byte,
) Packet {
	var out []byte
	out = append(out, nonce[:]...)
	out = box.Seal(out, plaintext, nonce, peersPublicKey, privateKey)

	return out
}

func (pkt Packet) ciphertext() []byte {
	return pkt[24:]
}

func (pkt Packet) open(peersPublicKey, privateKey *[32]byte) (boxPlaintext, bool) {
	return box.Open(nil, pkt.ciphertext(), pkt.nonce(), peersPublicKey, privateKey)
}

func (pkt Packet) valid() bool {
	return len(pkt) >= 24
}

type boxPlaintext []byte

func newBoxPlaintext(ratchetPublicKey *[32]byte, payload []byte) boxPlaintext {
	var out []byte
	out = append(out, ratchetPublicKey[:]...)
	out = append(out, payload...)
	return out
}

func (p boxPlaintext) ratchetPublicKey() *[32]byte {
	var ret [32]byte
	copy(ret[:], p[:32])
	return &ret
}

func (p boxPlaintext) payload() []byte {
	return p[32:]
}

func (p boxPlaintext) valid() bool {
	return len(p) >= 32
}

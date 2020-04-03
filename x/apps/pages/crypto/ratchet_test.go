package crypto

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

func TestBoxCurve25519(t *testing.T) {

	// golang.org/x/crypto/nacl/box should not stop using curve25519 for
	// its public and private keys for DHRatchet to work.

	publicKey, privateKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	expectedPubkey, err := curve25519.X25519(privateKey[:], curve25519.Basepoint)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expectedPubkey, publicKey[:])

}
func TestDHRatchet(t *testing.T) {

	publicKey1, privateKey1, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	publicKey2, privateKey2, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	var testMsgs [][]byte
	for i := 0; i < 6; i++ {
		testMsgs = append(testMsgs, []byte(fmt.Sprint("test msg", i)))
	}

	t.Run("roundtrip", func(t *testing.T) {

		alice := NewDHRatchet(publicKey2, privateKey1, true)
		bob := NewDHRatchet(publicKey1, privateKey2, false)

		m, ok, act := bob.Decrypt(alice.Encrypt(testMsgs[0]))
		assertEqual(t, testMsgs[0], true, RatchetAdvanced, m, ok, act)

		m, ok, act = alice.Decrypt(bob.Encrypt(testMsgs[1]))
		assertEqual(t, testMsgs[1], true, RatchetAdvanced, m, ok, act)

		m, ok, act = bob.Decrypt(alice.Encrypt(testMsgs[2]))
		assertEqual(t, testMsgs[2], true, RatchetAdvanced, m, ok, act)

		m, ok, act = alice.Decrypt(bob.Encrypt(testMsgs[3]))
		assertEqual(t, testMsgs[3], true, RatchetAdvanced, m, ok, act)

	})

	t.Run("stream of messages", func(t *testing.T) {

		alice := NewDHRatchet(publicKey2, privateKey1, true)
		bob := NewDHRatchet(publicKey1, privateKey2, false)

		buf := [][]byte{
			alice.Encrypt(testMsgs[0]),
			alice.Encrypt(testMsgs[1]),
			alice.Encrypt(testMsgs[2]),
		}

		m, ok, act := bob.Decrypt(buf[0])
		assertEqual(t, testMsgs[0], true, RatchetAdvanced, m, ok, act)
		m, ok, act = bob.Decrypt(buf[1])
		assertEqual(t, testMsgs[1], true, RatchetNoop, m, ok, act)
		m, ok, act = bob.Decrypt(buf[2])
		assertEqual(t, testMsgs[2], true, RatchetNoop, m, ok, act)

		buf = [][]byte{
			bob.Encrypt(testMsgs[3]),
			bob.Encrypt(testMsgs[4]),
			bob.Encrypt(testMsgs[5]),
		}

		m, ok, act = alice.Decrypt(buf[0])
		assertEqual(t, testMsgs[3], true, RatchetAdvanced, m, ok, act)
		m, ok, act = alice.Decrypt(buf[1])
		assertEqual(t, testMsgs[4], true, RatchetNoop, m, ok, act)
		m, ok, act = alice.Decrypt(buf[2])
		assertEqual(t, testMsgs[5], true, RatchetNoop, m, ok, act)

	})

	t.Run("concurrent messages after the ratchet is warmed should succeed", func(t *testing.T) {

		alice := NewDHRatchet(publicKey2, privateKey1, true)
		bob := NewDHRatchet(publicKey1, privateKey2, false)

		m, ok, act := bob.Decrypt(alice.Encrypt(testMsgs[0]))
		assertEqual(t, testMsgs[0], true, RatchetAdvanced, m, ok, act)

		p1 := alice.Encrypt(testMsgs[1])
		p2 := bob.Encrypt(testMsgs[2])

		m, ok, act = bob.Decrypt(p1)
		assertEqual(t, testMsgs[1], true, RatchetNoop, m, ok, act)
		m, ok, act = alice.Decrypt(p2)
		assertEqual(t, testMsgs[2], true, RatchetAdvanced, m, ok, act)

	})

	t.Run("compromised ratchet keys shouldn't decrypt packets from past or future steps", func(t *testing.T) {

		alice := NewDHRatchet(publicKey2, privateKey1, true)
		bob := NewDHRatchet(publicKey1, privateKey2, false)

		var comprPublicKey, comprPrivateKey *[32]byte

		var packets []Packet
		for i := 0; i < 3; i++ {

			if i == 1 {
				comprPublicKey, comprPrivateKey = alice.peersPublicKey, alice.privateKey
			}

			p := alice.Encrypt(testMsgs[0])
			_, ok, _ := bob.Decrypt(p)
			assert.True(t, ok)
			packets = append(packets, p)

			p = bob.Encrypt(testMsgs[0])
			_, ok, _ = alice.Decrypt(p)
			assert.True(t, ok)
			packets = append(packets, p)

		}

		eve := NewDHRatchet(comprPublicKey, comprPrivateKey, false)

		var decrypted []bool
		for _, p := range packets {
			_, ok1, _ := eve.Decrypt(p)
			newEve := NewDHRatchet(comprPublicKey, comprPrivateKey, false)
			_, ok2, _ := newEve.Decrypt(p)
			decrypted = append(decrypted, ok1 || ok2)
		}

		assert.Equal(t,
			[]bool{false, false, true, false, false, false},
			decrypted,
		)

	})

}

func assertEqual(t *testing.T, args ...interface{}) {
	if len(args)%2 != 0 {
		t.Fatal("bad number of arguments to assertEqual")
	}

	for i := 0; i < len(args)/2; i++ {
		assert.Equal(t, args[i], args[len(args)/2+i])
	}
}

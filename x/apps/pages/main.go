package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Gaboose/psycho"
	psychonats "github.com/Gaboose/psycho/servers/nats-adapter"

	"github.com/nats-io/nats.go"
)

type Button struct {
	psycho.Server
}

func (b *Button) HandleInfo(info map[string]interface{}) {
	fmt.Println(info)
}

func (b *Button) HandleMsg(subject string, payload []byte) {
	fmt.Printf("recv from %s: %s\n", subject, string(payload))
}

// https://leanpub.com/gocrypto/read#leanpub-auto-ed25519
// https://crypto.stackexchange.com/questions/68121/curve25519-over-ed25519-for-key-exchange-why
// https://github.com/golang/go/issues/20504
func (b *Button) Run() {

	b.Sub("mytopic")

	ticker := time.NewTicker(time.Second)
	for t := range ticker.C {
		b.Pub("mytopic", []byte(fmt.Sprint(t)))
	}
}

func main() {

	addr := flag.String("nats-server", "demo.nats.io:4222", "nats server to connect to")
	flag.Parse()

	// privateKey, err := pem.ParseX25519PrivateKey(bts)
	// curve25519.X25519(privateKey, curve25519.Basepoint)

	conn, err := nats.Connect(*addr)
	if err != nil {
		panic(err)
	}

	button := &Button{}
	button.Server = psychonats.New(conn)

	filename := flag.Arg(0)
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	reader := NewReadRateLimiter(f, 10*1024)

	var data [1024]byte
	for {
		n, err := reader.Read(data[:])
		if err != nil {
			panic(err)
		}
		button.Server.Pub("mysubject", data[:n])
	}
}

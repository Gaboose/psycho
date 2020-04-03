package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Gaboose/psycho"
	"github.com/Gaboose/psycho/servers"
)

func main() {

	natsBool := flag.Bool("n", true, "over nats")
	multicastBool := flag.Bool("m", false, "over multicast")
	natsAddr := flag.String("na", "demo.nats.io:4222", "nats server address")
	multicastAddr := flag.String("ma", "224.0.0.1:9999", "multicast group")
	multicastInterface := flag.String("mi", "wlp3s0", "multicast interface")

	receiverBool := flag.Bool("r", false, "receiver")
	flag.Parse()

	var server psycho.Server
	var err error

	switch {
	case *multicastBool:
		server, err = servers.NewMulticast(*multicastAddr, *multicastInterface)
	case *natsBool:
		server, err = servers.NewNATS(*natsAddr)
	default:
		flag.Usage()
		return
	}
	if err != nil {
		log.Println(err)
		return
	}

	// codec := psycho.NewServerCodec(os.Stdin, os.Stdout)

	if !*receiverBool {
		go func() {
			for i := 0; ; i++ {
				server.Pub("subject", []byte(strconv.Itoa(i)))
				time.Sleep(100 * time.Microsecond)
			}
		}()
	} else {
		server.Sub("subject")
	}

	// go codec.ServeClientOpsTo(server)
	server.ServeServerOpsTo(&handler{})
}

type handler struct {
	i int
}

func (h *handler) HandleInfo(info map[string]interface{}) {
	fmt.Println(info)
}

func (h *handler) HandleMsg(subject string, payload []byte) {
	i, err := strconv.Atoi(string(payload))
	if err != nil {
		fmt.Println(err)
	}

	if i == 0 {
		fmt.Println("first", i)
	} else if i-h.i != 1 {
		fmt.Println(h.i, "->", i)
	}

	h.i = i
}

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Gaboose/psycho"
)

type VersionString struct {
	ID      string
	Version uint64
	Payload string
}

func (vs *VersionString) Merge(version uint64, payload string) {
	if version < vs.Version {
		return
	}
	if strings.Compare(payload, vs.Payload) < 0 {
		return
	}
	vs.Payload = payload
}

func (vs *VersionString) Mutate(payload string) {
	vs.Version++
	vs.Payload = payload
}

func main() {
	flag.Parse()
	fname := flag.Arg(0)

	file, err := os.OpenFile(fname, os.O_RDWR, os.ModePerm)
	if err != nil {
		panic(err)
	}

	bts, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(bts), "\n")

	fmt.Println(len(lines), lines[0])

	p1, p2 := net.Pipe()

	client := psycho.NewClient(p1, p2)
	// fmt.Println(client.Info())

	fmt.Println("z")

	for {
		conn, err := client.Dial("mytopic")
		if err != nil {
			panic(err)
		}
		err = conn.Send([]byte("hello"))
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Second)
		fmt.Println("sent")
	}
}

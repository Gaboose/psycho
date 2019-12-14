package main

import (
	"context"
	"flag"
	"fmt"
	"os/exec"
	"time"

	"github.com/Gaboose/psoni"
)

func main() {
	server := flag.String("server", "", "psoni server binary")
	flag.Parse()

	cmd := exec.CommandContext(context.Background(), *server)
	serverOut, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	serverIn, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	client := psoni.NewClient(serverOut, serverIn)
	fmt.Println(client.Info())

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

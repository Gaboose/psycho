package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"

	"github.com/Gaboose/psycho"
	"github.com/Gaboose/psycho/servers"
)

func printInterfaceInfo(verbose bool) {

	ifaces, err := net.Interfaces()
	if err != nil {
		log.Println(err)
		return
	}

	data := [][]string{}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		addrsStr := []string{}
		for _, addr := range addrs {
			addrsStr = append(addrsStr, fmt.Sprint(addr))
		}
		if err != nil {
			addrsStr = []string{err.Error()}
		}

		maddrs, err := iface.MulticastAddrs()
		maddrsStr := []string{}
		for _, maddr := range maddrs {
			maddrsStr = append(maddrsStr, fmt.Sprint(maddr))
		}
		if err != nil {
			maddrsStr = []string{err.Error()}
		}

		row := []string{
			iface.Name,
			iface.Flags.String(),
			iface.HardwareAddr.String(),
			strings.Join(addrsStr, "\n"),
		}
		if verbose {
			row = append(row, strings.Join(maddrsStr, "\n"))
		}

		data = append(data, row)

	}

	table := tablewriter.NewWriter(os.Stdout)
	if verbose {
		table.SetHeader([]string{"Name", "Flags", "HW Address", "Addresses", "Multicast Groups"})
	} else {
		table.SetHeader([]string{"Name", "Flags", "HW Address", "Addresses"})
	}
	table.SetRowLine(true)

	for _, v := range data {
		table.Append(v)
	}
	table.Render() // Send output
}

func main() {

	natsBool := flag.Bool("n", true, "over nats")
	multicastBool := flag.Bool("m", false, "over multicast")
	natsAddr := flag.String("na", "demo.nats.io:4222", "nats server address")
	multicastAddr := flag.String("ma", "224.0.0.1:9999", "multicast group")
	multicastInterface := flag.String("mi", "wlp3s0", "multicast interface")

	infoInterfacesBool := flag.Bool("info", false, "print network interface information")
	verbose := flag.Bool("v", false, "verbose")
	flag.Parse()

	if *infoInterfacesBool {
		printInterfaceInfo(*verbose)
		return
	}

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

	codec := psycho.NewServerCodec(os.Stdin, os.Stdout)

	go codec.ServeClientOpsTo(server)
	server.ServeServerOpsTo(codec)
}

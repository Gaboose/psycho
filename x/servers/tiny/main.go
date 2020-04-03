package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/Gaboose/psycho"
)

type TinyServer struct {
	info map[string]interface{}
	subs map[string]map[chan<- *psycho.ClientOperation]struct{}
	mu   sync.Mutex
}

func NewTinyServer() *TinyServer {
	return &TinyServer{
		info: map[string]interface{}{
			"name":    "tiny",
			"version": "0.1",
		},
		subs: map[string]map[chan<- *psycho.ClientOperation]struct{}{},
	}
}

func (r *TinyServer) Serve(conn net.Conn) {
	defer conn.Close()
	clientOpCh := make(chan *psycho.ClientOperation, 10)
	go reader(psycho.NewServerDecoder(conn), clientOpCh)

	encoder := psycho.NewServerEncoder(conn)
	encoder.Info(r.info)

	recvMsgCh := make(chan *psycho.ClientOperation, 10)

	subscribedSubjects := map[string]struct{}{}
	defer func() {
		r.mu.Lock()
		for sub := range subscribedSubjects {
			delete(r.subs[sub], recvMsgCh)
		}
		r.mu.Unlock()
	}()

	for {
		select {
		case op, ok := <-clientOpCh:
			if !ok {
				return
			}
			if op.Error != nil {
				log.Printf("closing connection %v: %v", conn.RemoteAddr(), op.Error)
				encoder.Err(op.Error.Error())
				return
			}

			switch op.Type {
			case psycho.TypePublish:
				r.mu.Lock()
				chans := r.subs[op.Subject]
				r.mu.Unlock()
				for ch := range chans {
					select {
					case ch <- op:
					default:
						log.Printf("channel full on subject %v\n", op.Subject)
					}
				}
				encoder.OK()
			case psycho.TypeSubscribe:
				subscribedSubjects[op.Subject] = struct{}{}
				r.mu.Lock()
				m, ok := r.subs[op.Subject]
				if !ok {
					m = map[chan<- *psycho.ClientOperation]struct{}{}
					r.subs[op.Subject] = m
				}
				m[recvMsgCh] = struct{}{}
				r.mu.Unlock()
				encoder.OK()
			case psycho.TypeUnsubscribe:
				delete(subscribedSubjects, op.Subject)
				r.mu.Lock()
				delete(r.subs[op.Subject], recvMsgCh)
				r.mu.Unlock()
				encoder.OK()
			}
		case op := <-recvMsgCh:
			encoder.Msg(op.Subject, op.Payload)
		}
	}

}

func reader(dec *psycho.ServerDecoder, ch chan<- *psycho.ClientOperation) {
	for {
		op, ok := dec.ReadOperation()
		ch <- &op
		if !ok {
			close(ch)
			return
		}
	}
}

func main() {
	addr := flag.String("address", "localhost:5023", "address to listen on")
	flag.Parse()

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Print(err)
		return
	}
	fmt.Printf("listening on %v\n", *addr)

	tiny := NewTinyServer()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("error accepting a connection: %v", err)
			return
		}
		log.Printf("new connection %v", conn.RemoteAddr())

		go tiny.Serve(conn)
	}
}

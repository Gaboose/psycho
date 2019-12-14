package main

import (
	"flag"
	"io"
	"os"

	"github.com/Gaboose/psoni"
	nats "github.com/nats-io/nats.go"
)

type controller struct {
	conn    *nats.Conn
	enc     *psoni.ServerEncoder
	ops     <-chan *psoni.ClientOperation
	subs    map[string]*nats.Subscription
	subMsgs chan *nats.Msg
}

func newController(conn *nats.Conn, enc *psoni.ServerEncoder, ops <-chan *psoni.ClientOperation) *controller {
	return &controller{
		conn:    conn,
		enc:     enc,
		ops:     ops,
		subs:    map[string]*nats.Subscription{},
		subMsgs: make(chan *nats.Msg, 10),
	}
}

func (c *controller) subscribe(subject string) error {
	if _, ok := c.subs[subject]; ok {
		return nil
	}
	sub, err := c.conn.ChanSubscribe(subject, c.subMsgs)
	if err != nil {
		return err
	}
	c.subs[sub.Subject] = sub
	return nil
}

func (c *controller) run() {
	for {
		select {
		case op, ok := <-c.ops:
			if op == nil && ok {
				c.enc.Err("Parser Error")
				return
			} else if !ok {
				return
			}
			switch op.Type {
			case psoni.TypePublish:
				if err := c.subscribe(op.Subject); err != nil {
					c.enc.Err(err.Error())
					continue
				}
				c.enc.OKErr(c.conn.Publish(op.Subject, op.Payload))
			case psoni.TypeSubscribe:
				c.enc.OKErr(c.subscribe(op.Subject))
			case psoni.TypeUnsubscribe:
				sub, _ := c.subs[op.Subject]
				c.enc.OKErr(sub.Unsubscribe())
				delete(c.subs, op.Subject)
			}
		case msg := <-c.subMsgs:
			c.enc.Message(msg.Subject, msg.Data)
		}
	}
}

func reader(dec *psoni.ServerDecoder, ch chan<- *psoni.ClientOperation) {
	for {
		op, err := dec.ReadOperation()
		switch err {
		case io.EOF:
			return
		case nil:
		default:
			// parser error
			ch <- nil
			return
		}
		ch <- &op
	}
}

func main() {
	addr := flag.String("nats-server", "demo.nats.io:4222", "nats server to connect to")
	flag.Parse()

	decoder := psoni.NewServerDecoder(os.Stdin)
	encoder := psoni.NewServerEncoder(os.Stdout)
	encoder.Info(map[string]interface{}{
		"type":    "nats-adapter",
		"version": "0.1",
	})

	conn, err := nats.Connect(*addr)
	if err != nil {
		encoder.Err("Connection Error")
		return
	}
	defer conn.Close()

	clientOps := make(chan *psoni.ClientOperation, 10)
	controller := newController(conn, encoder, clientOps)
	go controller.run()
	reader(decoder, clientOps)
}

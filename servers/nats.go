package servers

import (
	"github.com/Gaboose/psycho"
	nats "github.com/nats-io/nats.go"
)

type NATS struct {
	subs  map[string]*nats.Subscription
	subCh chan *nats.Msg
	conn  *nats.Conn
}

func NewNATS(addr string) (*NATS, error) {
	conn, err := nats.Connect(addr, nats.NoEcho())
	if err != nil {
		return nil, err
	}

	return &NATS{
		conn:  conn,
		subs:  map[string]*nats.Subscription{},
		subCh: make(chan *nats.Msg, 64),
	}, nil
}

func (n *NATS) Pub(subject string, payload []byte) {
	n.conn.Publish(subject, payload)
}

func (n *NATS) Sub(subject string) {
	if _, ok := n.subs[subject]; ok {
		return
	}

	sub, err := n.conn.ChanSubscribe(subject, n.subCh)
	if err != nil {
		// TODO: notify client via some new logging type messages
		return
	}

	n.subs[subject] = sub

	return
}

func (n *NATS) Unsub(subject string) {
	sub, ok := n.subs[subject]
	if !ok {
		return
	}

	sub.Unsubscribe()
	delete(n.subs, subject)

	return
}

func (n *NATS) ServeServerOpsTo(client psycho.Client) {
	client.HandleInfo(map[string]interface{}{
		"type":    "nats",
		"version": "0.1",
	})
	for msg := range n.subCh {
		client.HandleMsg(msg.Subject, msg.Data)
	}
}

func (n *NATS) Close() {
	n.conn.Close()
}

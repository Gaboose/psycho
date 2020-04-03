package servers

import (
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/Gaboose/psycho"
	"golang.org/x/net/ipv4"
)

type Multicast struct {
	conn       *net.UDPConn
	packetConn *ipv4.PacketConn
	groupAddr  *net.UDPAddr
	bufferSize int

	subscribed map[string]struct{}
	nonces     *seenNonces

	client psycho.Client
}

func NewMulticast(group, iface string) (*Multicast, error) {
	groupAddr, err := net.ResolveUDPAddr("udp4", group)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: groupAddr.Port,
	})
	if err != nil {
		return nil, err
	}

	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, err
	}

	packetConn := ipv4.NewPacketConn(conn)
	packetConn.SetMulticastInterface(ifi)
	packetConn.SetControlMessage(ipv4.FlagDst, true)

	if err := packetConn.JoinGroup(ifi, groupAddr); err != nil {
		return nil, err
	}

	conn.SetReadBuffer(8192)

	return &Multicast{
		conn:       conn,
		packetConn: packetConn,
		groupAddr:  groupAddr,
		bufferSize: 8192,

		subscribed: map[string]struct{}{},
		nonces: &seenNonces{
			set: map[string]struct{}{},
			ttl: 10 * time.Second,
		},
	}, nil

}

func (m *Multicast) Pub(subject string, payload []byte) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		panic(err)
	}

	bts, err := json.Marshal(wireMsg{
		Subject: subject,
		Payload: payload,
		Nonce:   nonce,
	})
	if err != nil {
		panic(err)
	}

	m.nonces.Seen(string(nonce))

	_, err = m.packetConn.WriteTo(bts, nil, m.groupAddr)
	if err != nil {
		panic(err)
	}
}

func (m *Multicast) Sub(subject string) {
	m.subscribed[subject] = struct{}{}
}

func (m *Multicast) Unsub(subject string) {
	delete(m.subscribed, subject)
}

func (m *Multicast) ServeServerOpsTo(client psycho.Client) {
	client.HandleInfo(map[string]interface{}{
		"type":    "multicast",
		"version": "0.1",
	})
	buf := make([]byte, m.bufferSize)
	for {
		n, cm, _, err := m.packetConn.ReadFrom(buf)
		if err != nil {
			panic(err)
		}
		if !cm.Dst.Equal(m.groupAddr.IP) {
			continue
		}

		var msg wireMsg
		err = json.Unmarshal(buf[:n], &msg)
		if err != nil {
			log.Println(err)
			continue
		}

		if m.nonces.Seen(string(msg.Nonce)) {
			continue
		}

		if _, ok := m.subscribed[msg.Subject]; !ok {
			continue
		}

		client.HandleMsg(msg.Subject, msg.Payload)
	}
}

type wireMsg struct {
	Subject string
	Payload []byte
	Nonce   []byte
}

type seenNonces struct {
	set   map[string]struct{}
	slice []struct {
		time  time.Time
		nonce string
	}
	ttl time.Duration
	mu  sync.Mutex
}

func (n *seenNonces) Seen(nonce string) bool {
	now := time.Now()
	cutoff := now.Add(-n.ttl)

	n.mu.Lock()

	var i int
	for i = range n.slice {
		if n.slice[i].time.After(cutoff) {
			break
		}
	}
	n.slice = n.slice[i:]

	if _, ok := n.set[nonce]; ok {
		n.mu.Unlock()
		return true
	}

	n.set[nonce] = struct{}{}

	n.slice = append(n.slice, struct {
		time  time.Time
		nonce string
	}{now, nonce})

	n.mu.Unlock()

	return false
}

package psycho

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
)

type Client interface {
	HandleInfo(info map[string]interface{})
	HandleMsg(subject string, payload []byte)
}

type ErrConnClosed struct{}

func (e ErrConnClosed) Error() string { return "psycho: using a closed connection" }

type client struct {
	dec  *clientDecoder
	enc  *clientEncoder
	send chan *ClientOperation

	info  map[string]string
	conns map[string]map[*Conn]struct{}

	infoReceived chan struct{}
	closing      chan struct{}
	fatalErr     error
	sync.RWMutex
}

func Newclient(reader io.Reader, writer io.Writer) *client {
	c := &client{
		dec:  NewClientDecoder(reader),
		enc:  NewClientEncoder(writer),
		send: make(chan *ClientOperation),

		conns: map[string]map[*Conn]struct{}{},

		infoReceived: make(chan struct{}),
		closing:      make(chan struct{}),
	}
	go c.reader()
	go c.writer()
	return c
}

func (c *client) Dial(subject string) (*Conn, error) {
	select {
	case c.send <- &ClientOperation{
		Type:    TypeSubscribe,
		Subject: subject,
	}:
	case <-c.closing:
		return nil, ErrConnClosed{}
	}
	conn := &Conn{
		subject: subject,
		recv:    make(chan []byte),
		send:    c.send,
	}
	c.Lock()
	if _, ok := c.conns[subject]; !ok {
		c.conns[subject] = map[*Conn]struct{}{}
	}
	c.conns[subject][conn] = struct{}{}
	c.Unlock()
	return conn, nil
}

func (c *client) Info() (map[string]string, error) {
	select {
	case <-c.infoReceived:
	case <-c.closing:
		return nil, c.fatalErr
	}
	return c.info, nil
}

func (c *client) unsubscribe(conn *Conn) {
	c.Lock()
	delete(c.conns[conn.subject], conn)
	c.Unlock()
	close(conn.recv)
	close(conn.closing)
}

func (c *client) reader() {
	var infoOnce sync.Once
	var conns []*Conn
	for {
		op, err := c.dec.ReadOperation()
		if err != nil {
			c.fatal(err)
			return
		}
		switch op.Type {
		case TypeMessage:
			conns = conns[:0]
			c.RLock()
			for conn := range c.conns[op.Subject] {
				conns = append(conns, conn)
			}
			c.RUnlock()
			for _, conn := range conns {
				conn.received(op.Payload)
			}
		case TypeInfo:
			infoOnce.Do(func() {
				c.info = op.Map
				close(c.infoReceived)
			})
		case TypeOK:
		case TypeError:
			c.fatal(errors.New(string(op.Payload)))
			return
		}
	}
}

func (c *client) writer() {
	for op := range c.send {
		switch op.Type {
		case TypePublish:
			c.enc.Publish(op.Subject, op.Payload)
		case TypeSubscribe:
			c.enc.Subscribe(op.Subject)
		case TypeUnsubscribe:
			c.enc.Unsubscribe(op.Subject)
		}
	}
}

func (c *client) fatal(err error) {
	c.fatalErr = err
	close(c.closing)
}

type Conn struct {
	subject string
	recv    chan []byte
	send    chan *ClientOperation

	client *client

	recvMsgs, recvBytes uint64
	closing             chan struct{}
}

func (c *Conn) Send(payload []byte) error {
	select {
	case c.send <- &ClientOperation{
		Type:    TypePublish,
		Subject: c.subject,
		Payload: payload,
	}:
		return nil
	case <-c.closing:
		return ErrConnClosed{}
	}
}

func (c *Conn) Receive() ([]byte, error) {
	payload, ok := <-c.recv
	if !ok {
		return nil, ErrConnClosed{}
	}
	return payload, nil
}

func (c *Conn) Close() {
	c.client.unsubscribe(c)
}

func (c *Conn) received(payload []byte) {
	atomic.AddUint64(&c.recvMsgs, 1)
	atomic.AddUint64(&c.recvBytes, uint64(len(payload)))
	select {
	case c.recv <- payload:
	default:
	}
}

type ServerOpType int

const (
	TypeInfo ServerOpType = iota + 1
	TypeMessage
	TypeOK
	TypeError
)

type ServerOperation struct {
	Type    ServerOpType
	Subject string
	Payload []byte
	Map     map[string]string
}

type clientDecoder struct {
	reader *bufio.Reader
}

func NewClientDecoder(r io.Reader) *clientDecoder {
	return &clientDecoder{
		reader: bufio.NewReader(r),
	}
}

func (d *clientDecoder) ReadOperation() (ServerOperation, error) {
	line, err := d.reader.ReadString('\n')
	if err != nil {
		return ServerOperation{}, err
	}
	tokens := strings.Split(line[:len(line)-1], " ")

	if len(tokens) < 1 {
		return ServerOperation{}, errors.New("len(tokens) < 1")
	}

	switch tokens[0] {
	case "INFO":
		if len(tokens) != 2 {
			return ServerOperation{}, errors.New("INFO len(tokens) != 2")
		}
		m := map[string]json.Number{}
		if err := json.Unmarshal([]byte(tokens[1]), &m); err != nil {
			return ServerOperation{}, err
		}
		op := ServerOperation{
			Type:    TypeInfo,
			Payload: []byte(tokens[1]),
			Map:     map[string]string{},
		}
		for k, v := range m {
			op.Map[k] = string(v)
		}
		return op, nil
	case "MSG":
		if len(tokens) != 3 {
			return ServerOperation{}, errors.New("MSG len(tokens) != 3")
		}
		payload, err := readPayload(d.reader, tokens[2])
		if err != nil {
			return ServerOperation{}, ErrParser{}
		}
		return ServerOperation{
			Type:    TypeMessage,
			Subject: tokens[1],
			Payload: payload,
		}, nil
	case "+OK":
		return ServerOperation{
			Type: TypeOK,
		}, nil
	case "-ERR":
		if len(tokens) != 2 {
			return ServerOperation{}, errors.New("ERR len(tokens) != 2")
		}
		return ServerOperation{
			Type:    TypeError,
			Payload: []byte(strings.Trim(tokens[1], "'")),
		}, nil
	}
	return ServerOperation{}, ErrParser{}
}

type clientEncoder struct {
	writer io.Writer
}

func NewClientEncoder(writer io.Writer) *clientEncoder {
	return &clientEncoder{
		writer: writer,
	}
}

func (e *clientEncoder) Publish(subject string, payload []byte) error {
	_, err := fmt.Fprintf(e.writer, "PUB %s %d\n%s\n", subject, len(payload), string(payload))
	return err
}

func (e *clientEncoder) Subscribe(subject string) error {
	_, err := fmt.Fprintf(e.writer, "SUB %s\n", subject)
	return err
}

func (e *clientEncoder) Unsubscribe(subject string) error {
	_, err := fmt.Fprintf(e.writer, "UNSUB %s\n", subject)
	return err
}

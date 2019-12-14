package psoni

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type ErrParser struct{}

func (e ErrParser) Error() string { return "Parser Error" }

type ClientOpType int

const (
	TypeSubscribe ClientOpType = iota + 1
	TypeUnsubscribe
	TypePublish
)

type ClientOperation struct {
	Type    ClientOpType
	Subject string
	Payload []byte
}

type ServerDecoder struct {
	reader *bufio.Reader
}

func NewServerDecoder(r io.Reader) *ServerDecoder {
	return &ServerDecoder{
		reader: bufio.NewReader(r),
	}
}

func (p *ServerDecoder) ReadOperation() (ClientOperation, error) {
	line, err := p.reader.ReadString('\n')
	if err != nil {
		return ClientOperation{}, err
	}
	tokens := strings.Split(line[:len(line)-1], " ")

	if len(tokens) < 2 {
		return ClientOperation{}, errors.New("len(tokens) < 2")
	}

	switch tokens[0] {
	case "SUB":
		return ClientOperation{
			Type:    TypeSubscribe,
			Subject: tokens[1],
		}, nil
	case "UNSUB":
		return ClientOperation{
			Type:    TypeUnsubscribe,
			Subject: tokens[1],
		}, nil
	case "PUB":
		if len(tokens) != 3 {
			return ClientOperation{}, errors.New("PUB; len(tokens) != 3")
		}
		payload, err := readPayload(p.reader, tokens[2])
		if err != nil {
			return ClientOperation{}, ErrParser{}
		}
		return ClientOperation{
			Type:    TypePublish,
			Subject: tokens[1],
			Payload: payload,
		}, nil
	}
	return ClientOperation{}, ErrParser{}
}

type ServerEncoder struct {
	writer io.Writer
}

func NewServerEncoder(w io.Writer) *ServerEncoder {
	return &ServerEncoder{
		writer: w,
	}
}

func (e *ServerEncoder) Info(values map[string]interface{}) error {
	bts, err := json.Marshal(values)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(e.writer, "INFO %s\n", string(bts))
	return err
}

func (e *ServerEncoder) Message(subject string, payload []byte) error {
	_, err := fmt.Fprintf(e.writer, "MSG %s %d\n%s\n", subject, len(payload), string(payload))
	return err
}

func (e *ServerEncoder) OK() error {
	_, err := e.writer.Write([]byte("+OK\n"))
	return err
}

func (e *ServerEncoder) Err(message string) error {
	_, err := fmt.Fprintf(e.writer, "-ERR '%s'\n", message)
	return err
}

func (e *ServerEncoder) OKErr(err error) {
	if err == nil {
		e.OK()
		return
	}
	e.Err(err.Error())
}

func readPayload(reader *bufio.Reader, nbytes string) ([]byte, error) {
	n, err := strconv.ParseUint(nbytes, 10, 64)
	if err != nil {
		return nil, ErrParser{}
	}
	var payload = make([]byte, n)
	nread, err := reader.Read(payload)
	if nread < int(n) {
		return nil, ErrParser{}
	}
	if err != nil {
		return nil, ErrParser{}
	}
	delim, err := reader.ReadString('\n')
	if delim != "\n" {
		return nil, ErrParser{}
	}
	if err != nil {
		return nil, ErrParser{}
	}
	return payload, nil
}

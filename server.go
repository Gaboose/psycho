package psycho

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Server interface {
	Pub(subject string, payload []byte)
	Sub(subject string)
	Unsub(subject string)
	ServeServerOpsTo(client Client)
}

type ErrParser struct {
	reason string
}

func (e ErrParser) Error() string {
	return fmt.Sprintf("parser error: %s", e.reason)
}

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
	Error   error
}

type ServerDecoder struct {
	reader *bufio.Reader
}

func NewServerDecoder(r io.Reader) *ServerDecoder {
	return &ServerDecoder{
		reader: bufio.NewReader(r),
	}
}

func (p *ServerDecoder) ReadOperation() (ClientOperation, bool) {
	line, err := p.reader.ReadString('\n')
	if err == io.EOF {
		return ClientOperation{Error: io.EOF}, false
	} else if err != nil {
		return ClientOperation{Error: ErrParser{"reading new line"}}, false
	}
	tokens := strings.Split(line[:len(line)-1], " ")

	if len(tokens) < 2 {
		return ClientOperation{Error: ErrParser{
			fmt.Sprintf("expected 2 or more tokens, found %d", len(tokens)),
		}}, false
	}

	switch tokens[0] {
	case "SUB":
		if len(tokens) != 2 {
			return ClientOperation{Error: ErrParser{
				fmt.Sprintf("SUB op expects exactly 1 argument, found %d", len(tokens)-1),
			}}, false
		}
		return ClientOperation{
			Type:    TypeSubscribe,
			Subject: tokens[1],
		}, true
	case "UNSUB":
		if len(tokens) != 2 {
			return ClientOperation{Error: ErrParser{
				fmt.Sprintf("UNSUB op expects exactly 1 argument, found %d", len(tokens)-1),
			}}, false
		}
		return ClientOperation{
			Type:    TypeUnsubscribe,
			Subject: tokens[1],
		}, true
	case "PUB":
		if len(tokens) != 3 {
			return ClientOperation{Error: ErrParser{
				fmt.Sprintf("PUB op expects exactly 2 arguments, found %d", len(tokens)-1),
			}}, false
		}
		payload, err := readPayload(p.reader, tokens[2])
		if err != nil {
			return ClientOperation{Error: ErrParser{fmt.Sprintf("reading payload: %v", err)}}, false
		}
		return ClientOperation{
			Type:    TypePublish,
			Subject: tokens[1],
			Payload: payload,
		}, true
	default:
		return ClientOperation{Error: ErrParser{"unknown op name"}}, false
	}
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

func (e *ServerEncoder) Msg(subject string, payload []byte) error {
	_, err := fmt.Fprintf(e.writer, "MSG %s %d\n%s\n", subject, len(payload), string(payload))
	return err
}

func (e *ServerEncoder) OK() error {
	_, err := e.writer.Write([]byte("+OK\n"))
	return err
}

func (e *ServerEncoder) Err(message string) error {
	_, err := fmt.Fprintf(e.writer, "-ERR %q\n", message)
	return err
}

func (e *ServerEncoder) OKErr(err error) {
	if err == nil {
		e.OK()
		return
	}
	e.Err(err.Error())
}

// func readPayload(reader *bufio.Reader, nbytes string) ([]byte, error) {
// 	n, err := strconv.ParseUint(nbytes, 10, 64)
// 	if err != nil {
// 		return nil, errors.New("parsing number of bytes")
// 	}
// 	var payload = make([]byte, n)
// 	nread, err := reader.Read(payload)
// 	if nread < int(n) {
// 		return nil, fmt.Errorf("expected to read %d bytes, but only read %d", n, nread)
// 	}
// 	if err != nil {
// 		return nil, err
// 	}
// 	delim, err := reader.ReadString('\n')
// 	if delim != "\n" {
// 		return nil, fmt.Errorf("payload did not end with a new line")
// 	}
// 	if err != nil {
// 		return nil, err
// 	}
// 	return payload, nil
// }

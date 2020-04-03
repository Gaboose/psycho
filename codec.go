package psycho

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type ServerCodec struct {
	reader *bufio.Reader
	writer io.Writer
}

func NewServerCodec(reader io.Reader, writer io.Writer) *ServerCodec {
	c := &ServerCodec{
		reader: bufio.NewReader(reader),
		writer: writer,
	}
	return c
}

func (c *ServerCodec) HandleInfo(info map[string]interface{}) {
	m, err := json.Marshal(info)
	if err != nil {
		c.writer.Write([]byte(`INFO {"error": "error marshalling info"}\n`))
		return
	}
	fmt.Fprintf(c.writer, "INFO %s\n", m)
}

func (c *ServerCodec) HandleMsg(subject string, payload []byte) {
	buf := bytes.NewBuffer(make([]byte, 0, len(payload)+len(subject)+16))
	fmt.Fprintf(buf, "MSG %s %d\n", subject, len(payload))
	buf.Write(payload)
	buf.WriteRune('\n')
	c.writer.Write(buf.Bytes())
}

func (c *ServerCodec) ServeClientOpsTo(server Server) {
	for {
		op, subject, payload, err := c.readMsg()
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(c.writer, "-ERR %q\n", err.Error())
			continue
		}
		switch op {
		case pub:
			server.Pub(subject, payload)
		case sub:
			server.Sub(subject)
		case unsub:
			server.Unsub(subject)
		}
		c.writer.Write([]byte("+OK\n"))
	}
}

type opname string

const (
	pub   opname = "PUB"
	sub   opname = "SUB"
	unsub opname = "UNSUB"
)

func (c *ServerCodec) readLine() ([]byte, error) {
	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	line = line[:len(line)-1]
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return line, nil
}

func (c *ServerCodec) readMsg() (opname, string, []byte, error) {

	line, err := c.readLine()
	if err != nil {
		return "", "", nil, err
	}

	if len(line) == 0 {
		return "", "", nil, errors.New("empty line")
	}

	tokens := strings.Split(string(line), " ")

	switch opname(tokens[0]) {
	case pub:
		if len(tokens) != 3 {
			return "", "", nil, fmt.Errorf("PUB op expects exactly 2 arguments, found %d", len(tokens)-1)
		}
		payload, err := readPayload(c.reader, tokens[2])
		return pub, tokens[1], payload, err
	case sub:
		if len(tokens) != 2 {
			return "", "", nil, fmt.Errorf("SUB op expects exactly 1 argument, found %d", len(tokens)-1)
		}
		return sub, tokens[1], nil, nil
	case unsub:
		if len(tokens) != 2 {
			return "", "", nil, fmt.Errorf("UNSUB op expects exactly 1 argument, found %d", len(tokens)-1)
		}
		return unsub, tokens[1], nil, nil
	default:
		return "", "", nil, errors.New("unknown op name")
	}
}

func readPayload(reader *bufio.Reader, nbytes string) ([]byte, error) {
	n, err := strconv.ParseUint(nbytes, 10, 64)
	if err != nil {
		return nil, errors.New("parsing number of bytes")
	}
	var payload = make([]byte, n)
	nread, err := reader.Read(payload)
	if nread < int(n) {
		return nil, fmt.Errorf("expected to read %d bytes, but only read %d", n, nread)
	}
	if err != nil {
		return nil, err
	}
	delim, err := reader.ReadString('\n')
	if !(delim == "\n" || delim == "\r\n") {
		return nil, fmt.Errorf("payload did not end with a new line")
	}
	if err != nil {
		return nil, err
	}
	return payload, nil
}

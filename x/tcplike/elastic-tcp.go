package congestion

import "io"

type ElasticTCP struct {
	writer io.Writer
	beta   float64

	slowStart bool
	seq       int
	acked     int
	nDupAck   int

	ccwd int
}

func NewElasticTCP(writer io.Writer, beta float64) *ElasticTCP {
	return &ElasticTCP{writer: writer}
}

func (e *ElasticTCP) Write(p []byte) (n int, err error) {
	return e.writer.Write(p)
}

func (e *ElasticTCP) Seq() int {
	return e.seq
}

func (e *ElasticTCP) Ack(seq int) {
	if e.acked == seq {
		e.nDupAck++
	} else {
		e.nDupAck = 0
	}

	if e.nDupAck < 3 {
		if e.slowStart {
			e.ccwd++
		} else {

		}
	} else {
		e.ccwd *= e.beta
	}
}

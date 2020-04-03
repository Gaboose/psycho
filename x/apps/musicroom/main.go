package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"time"

	"golang.org/x/mobile/exp/audio/al"
)

var bluesScale = []int{0, 3, 5, 6, 7, 10}

func main() {
	// server := flag.String("server", "", "psycho server binary")
	// flag.Parse()

	// cmd := exec.CommandContext(context.Background(), *server)
	// serverOut, err := cmd.StdoutPipe()
	// if err != nil {
	// 	panic(err)
	// }
	// serverIn, err := cmd.StdinPipe()
	// if err != nil {
	// 	panic(err)
	// }
	// if err := cmd.Start(); err != nil {
	// 	panic(err)
	// }

	// client := psycho.NewClient(serverOut, serverIn)
	// fmt.Println(client.Info())

	// for {
	// 	conn, err := client.Dial("mytopic")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	err = conn.Send([]byte("hello"))
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	time.Sleep(time.Second)
	// 	fmt.Println("sent")
	// }

	al.OpenDevice()
	bufs := al.GenBuffers(2)
	srcs := al.GenSources(1)

	bts := sine(time.Second, 0.0)
	bufs[0].BufferData(al.FormatMono16, bts, freq)
	bufs[1].BufferData(al.FormatMono16, bts, freq)
	srcs[0].QueueBuffers(bufs[0])
	srcs[0].QueueBuffers(bufs[1])
	// bufs[1], bufs[0] = bufs[0], bufs[1]
	al.PlaySources(srcs[0])
	fmt.Println("playing", al.DeviceError(), len(bts))
	for {
		fmt.Println("a", srcs[0].BuffersQueued(), srcs[0].BuffersProcessed(), srcs[0].Gain(), al.DeviceError())
		fmt.Println("b", srcs[0].OffsetSeconds(), al.Renderer(), al.ListenerGain(), al.GetString(int(al.Error())))
		fmt.Println("offset", srcs[0].OffsetByte())
		time.Sleep(time.Second / 10)
		if srcs[0].BuffersQueued()-srcs[0].BuffersProcessed() <= 1 {
			srcs[0].UnqueueBuffers(bufs[0])
			bts := sine(time.Second/2, rand.Float64()-0.5)
			bufs[0].BufferData(al.FormatMono16, bts, freq)
			srcs[0].QueueBuffers(bufs[0])
			bufs[1], bufs[0] = bufs[0], bufs[1]
		}
	}
}

func note(dur time.Duration, frequency int) []byte {
	samples := uint64(dur * freq / time.Second)
	ret := make([]byte, samples*2)
	for i := uint64(0); i < samples; i++ {
		t := float64(i) / freq
		f := math.Sin((400 * (1 + factor*t) * t * (2 * math.Pi)))
		// ret[i] = uint8(f * math.MaxUint8)
		binary.LittleEndian.PutUint16(ret[i*2:], uint16(f*math.MaxUint16))
	}
	return ret
}

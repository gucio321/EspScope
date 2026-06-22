package network

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/kpango/glg"
)

const Magic = 0x53434F50 // "SCOP"

type Header struct {
	Magic       uint32
	Seq         uint32
	SampleRate  uint32
	SampleCount uint16
}

func Listen(udpPort int, samplesOut chan uint16, end chan bool) error {
	addr := net.UDPAddr{
		Port: udpPort,
		IP:   net.IPv4zero,
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return fmt.Errorf("initializing UDP connection: %w", err)
	}

	defer conn.Close()

	glg.Infof("Listening on UDP :%d\n", udpPort)

	buf := make([]byte, 2048)

	var lastSeq uint32
	firstPacket := true

	go func() {
		<-end
		conn.Close()
	}()

listening:
	for {
		n, remote, err := conn.ReadFromUDP(buf)
		switch {
		case errors.Is(err, net.ErrClosed):
			glg.Debug("Connection closed. Exitting.")
			return nil
		case err != nil:
			continue listening
		}

		// We need at least 14 bytes for the header
		if n < 14 {
			glg.Errorf("packet too short from %v: got=%d expected>=14\n", remote, n)
			continue
		}

		magic := binary.LittleEndian.Uint32(buf[0:4])
		if magic != Magic {
			glg.Errorf("invalid magic from %v: 0x%08X\n", remote, magic)
			continue
		}

		seq := binary.LittleEndian.Uint32(buf[4:8])
		// sampleRate := binary.LittleEndian.Uint32(buf[8:12])
		sampleCount := binary.LittleEndian.Uint16(buf[12:14])

		expectedSize := 14 + int(sampleCount)*2
		if n < expectedSize {
			glg.Errorf("short packet from %v: got=%d expected=%d\n", remote, n, expectedSize)
			continue
		}

		if !firstPacket {
			if seq != lastSeq+1 {
				glg.Warnf(
					"packet loss: expected=%d got=%d lost=%d",
					lastSeq+1,
					seq,
					seq-lastSeq-1,
				)
			}
		}

		firstPacket = false
		lastSeq = seq

		offset := 14
		for i := 0; i < int(sampleCount); i++ {
			samplesOut <- binary.LittleEndian.Uint16(
				buf[offset+i*2 : offset+i*2+2],
			)
		}
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

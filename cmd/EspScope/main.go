package main

import (
	"net"

	"github.com/gucio321/EspScope/pkg/network"
	"github.com/kpango/glg"
)

func main() {
	end := make(chan bool)
	data := make(chan uint16, 1024)
	go func() {
		if err := network.Listen(
			net.IPv4(127, 0, 0, 1),
			5000,
			data,
			end,
		); err != nil {
			glg.Fatalf("Failed to listen: %v", err)
		}
	}()

	for p := range data {
		glg.Infof("Received: %d", p)
	}
}

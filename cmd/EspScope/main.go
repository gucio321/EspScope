package main

import (
	"github.com/gucio321/EspScope/pkg/gui"
	"github.com/kpango/glg"
)

func main() {
	a := gui.New()
	if err := a.Run(); err != nil {
		glg.Fatalf("Unable to run app %v", err)
	}
	/*
		end := make(chan bool)
		data := make(chan uint16, 1024)
		go func() {
			if err := network.Listen(
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
	*/
}

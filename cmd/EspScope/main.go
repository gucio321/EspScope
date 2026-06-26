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
}

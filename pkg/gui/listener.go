package gui

import (
	"github.com/gucio321/EspScope/pkg/network"
	"github.com/kpango/glg"
)

const port = 5000

func (a *App) startListen() {
	go func() {
		glg.Info("Starting UDP Listen.")
		a.mutex.Lock()
		a.listening = true
		a.mutex.Unlock()

		if err := network.Listen(port, a.listener.data, a.listener.end); err != nil {
			a.mutex.Lock()
			a.listening = false
			a.mutex.Unlock()

			glg.Errorf("Error while trying to initialize listening on port %d: %v", port, err)
		}

		a.mutex.Lock()
		a.listening = false
		a.mutex.Unlock()
	}()
}

func (a *App) stopListen() {
	glg.Info("Stopping UDP listening...")
	a.listener.end <- true
	glg.Info("UDP Listening stopped")
}

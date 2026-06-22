package espmanager

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/kpango/glg"
)

var ErrInvalidResponse = errors.New("invalid server response received")

type ESPManager struct {
	ip net.IP
}

func New(ip net.IP) *ESPManager {
	return &ESPManager{
		ip: ip,
	}
}

func (m *ESPManager) StartStream() error {
	url := fmt.Sprintf("http://%s/start", m.ip)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("unable to access %s: %v", url, err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("invalid response for URL %s (expected http status 200, but got %d): %v", url, resp.StatusCode, ErrInvalidResponse)
	}

	glg.Debug("Stream started.")

	return nil
}

func (m *ESPManager) StopStream() error {
	url := fmt.Sprintf("http://%s/stop", m.ip)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("unable to access %s: %v", url, err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("invalid response for URL %s (expected http status 200, but got %d): %v", url, resp.StatusCode, ErrInvalidResponse)
	}

	glg.Debug("Stream stopped.")

	return nil
}

func (m *ESPManager) EndConnection() error {
	return m.StopStream()
}

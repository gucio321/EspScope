package espmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

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

	client := &http.Client{
		Timeout: time.Millisecond * 500,
	}

	resp, err := client.Get(url)
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

	client := &http.Client{
		Timeout: time.Millisecond * 500,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("unable to access %s: %v", url, err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("invalid response for URL %s (expected http status 200, but got %d): %v", url, resp.StatusCode, ErrInvalidResponse)
	}

	glg.Debug("Stream stopped.")

	return nil
}

func (m *ESPManager) SetFrequency(freq int) error {
	url := fmt.Sprintf("http://%s/sample_rate?set=%d", m.ip, freq)

	client := &http.Client{
		Timeout: time.Millisecond * 500,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("unable to access %s: %v", url, err)
	}

	switch resp.StatusCode {
	case 200:
		glg.Debug("Sample rate updated.")
		return nil
	case 400: // Bad request
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("bad request received while trying access %s but unable to read response %v", url, err)
		}

		var msg struct {
			Message string
		}

		if err := json.Unmarshal(body, &msg); err != nil {
			return fmt.Errorf("bad request received while trying access %s but unable to parse error message %v", url, err)
		}

		return fmt.Errorf("bad request: %v", msg.Message)
	default:
		return fmt.Errorf("invalid response for URL %s (expected http status 200, but got %d): %v", url, resp.StatusCode, ErrInvalidResponse)
	}
}

func (m *ESPManager) EndConnection() error {
	return m.StopStream()
}

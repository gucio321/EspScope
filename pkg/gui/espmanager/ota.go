package espmanager

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/kpango/glg"
)

func (m *ESPManager) OTAUpdate(filepath string) error {
	url := fmt.Sprintf("http://%s/ota", m.ip)

	stat, err := os.Stat(filepath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", filepath, err)
	}

	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("open %s: %w", filepath, err)
	}

	defer file.Close()

	req, err := http.NewRequest("POST", url, file)
	if err != nil {
		return fmt.Errorf("creating http request to %s: %w", url, err)
	}

	req.ContentLength = stat.Size()
	req.Header.Set("Content-Type", "application/octet-stream")

	client := http.Client{
		Timeout: 0,
	}

	glg.Debugf("Uploading firmware (%v bytes)", stat.Size())

	start := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request execution after %v: %w", time.Since(start), err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("invalid response for URL %s (expected http status 200, but got %d): %v", url, resp.StatusCode, ErrInvalidResponse)
	}

	glg.Debugf("Update successfull! Uploaded %v bytes in %v s", stat.Size(), time.Since(start))

	return nil
}

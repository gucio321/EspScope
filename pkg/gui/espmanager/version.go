package espmanager

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kpango/glg"
)

type VersionInfo struct {
	APIVersionMajor int
	APIVersionMinor int
	CommitID        string
	BuildDate       string
}

func (m *ESPManager) Version() (*VersionInfo, error) {
	result := &VersionInfo{}
	url := fmt.Sprintf("http://%s/version", m.ip)

	client := &http.Client{
		Timeout: time.Millisecond * 500,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("unable to access %s: %v", url, err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("invalid response for URL %s (expected http status 200, but got %d): %v", url, resp.StatusCode, ErrInvalidResponse)
	}

	defer resp.Body.Close()

	versionInfoString, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body for URL %s: %v", url, err)
	}

	glg.Debugf("Got version info string: %s", versionInfoString)

	if err := json.Unmarshal(versionInfoString, result); err != nil {
		return nil, fmt.Errorf("unable to unmarshal version info string for: %v", err)
	}

	return result, nil
}

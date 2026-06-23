package espmanager

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type LookupTable struct {
	data []uint16
}

func NewLookupTable() *LookupTable {
	return &LookupTable{}
}

func (m *ESPManager) GetLookupTable() (*LookupTable, error) {
	result := NewLookupTable()

	url := fmt.Sprintf("http://%s/adc_table", m.ip)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("unable to access %s: %v", url, err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("invalid response for URL %s (expected http status 200, but got %d): %v", url, resp.StatusCode, ErrInvalidResponse)
	}

	var table []uint16 = make([]uint16, 0, 4096)

reading:
	for {
		var chunk uint16
		err := binary.Read(resp.Body, binary.LittleEndian, &chunk)
		switch {
		case errors.Is(err, io.EOF):
			break reading
		case err != nil:
			return nil, fmt.Errorf("decoding table: %v", err)
		}

		table = append(table, chunk)
	}

	result.data = table

	return result, nil
}

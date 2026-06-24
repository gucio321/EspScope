package espmanager

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type LookupTable struct {
	data     []float64
	tableLen uint16
}

func NewLookupTable() *LookupTable {
	return &LookupTable{}
}

func (l *LookupTable) ADCToV(adc uint16) float64 {
	if adc >= l.tableLen {
		return 0
	}

	return l.data[adc]
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

	var table []float64 = make([]float64, 0, 4096)

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

		table = append(table, float64(chunk)/1000) // convert mV -> V
	}

	result.data = table
	result.tableLen = uint16(len(table))

	return result, nil
}

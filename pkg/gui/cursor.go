package gui

import "math"

type Cursor struct {
	Enabled bool
	Value   int
}

func (c *Cursor) Toggle(value float64) {
	c.Enabled = !c.Enabled
	if c.Enabled {
		c.Value = int(math.Round(value))
	}
}

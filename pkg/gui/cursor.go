package gui

import "math"

type Cursor struct {
	Enabled bool
	Value   int
}

func (c *Cursor) Toggle(value float64) {
	c.Enabled = !c.Enabled
	if c.Enabled {
		c.SetValue(value)
	}
}

func (c *Cursor) SetValue(value float64) {
	c.Value = int(math.Round(value))
}

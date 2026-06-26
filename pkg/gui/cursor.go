package gui

import "math"

type Cursor struct {
	Enabled bool
	Value   int
}

func (c *Cursor) Toggle(value float64) {
	if c.Enabled && c.Value == c.floatToInt(value) {
		c.Enabled = false

		return
	}

	c.Enabled = true
	c.SetValue(value)
}

func (c *Cursor) SetValue(value float64) {
	c.Value = c.floatToInt(value)
}

func (c *Cursor) floatToInt(v float64) int {
	return int(math.Round(v))
}

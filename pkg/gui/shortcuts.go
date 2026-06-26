package gui

import (
	"github.com/AllenDang/giu"
)

func (a *App) keyboardShortcuts() []giu.WindowShortcut {
	return []giu.WindowShortcut{
		{
			Key:      giu.KeyF,
			Modifier: giu.ModNone,
			Callback: func() {
				a.freez = !a.freez
			},
		},
		{
			Key:      giu.KeyA,
			Modifier: giu.ModNone,
			Callback: func() {
				a.cursorA.Toggle(a.mouseX)
			},
		},
		{
			Key:      giu.KeyB,
			Modifier: giu.ModNone,
			Callback: func() {
				a.cursorB.Toggle(a.mouseX)
			},
		},
	}
}

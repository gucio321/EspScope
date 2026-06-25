package gui

import "github.com/AllenDang/giu"

func (a *App) keyboardShortcuts() []giu.WindowShortcut {
	return []giu.WindowShortcut{
		{
			Key:      giu.KeyF,
			Modifier: giu.ModNone,
			Callback: func() {
				a.freez = !a.freez
			},
		},
	}
}

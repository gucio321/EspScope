package gui

import (
	"github.com/AllenDang/giu"
	"github.com/pkg/browser"
)

func (a *App) menuBar() giu.Widget {
	return giu.Layout{
		giu.Menu("File").Layout(a.fileMenu()),
		giu.Menu("Help").Layout(a.helpMenu()),
	}
}

func (a *App) fileMenu() giu.Widget {
	return giu.Layout{
		giu.MenuItem("Quit").OnClick(a.Quit),
	}
}

func (a *App) helpMenu() giu.Widget {
	return giu.Layout{
		giu.MenuItem("GitHub").OnClick(func() {
			browser.OpenURL(GHURL)
		}),
	}
}

package gui

import (
	"github.com/AllenDang/giu"
	"github.com/pkg/browser"
)

const frequencyPopupID = "FrequencyPopup"

func (a *App) menuBar() giu.Widget {
	return giu.Layout{
		giu.Menu("App").Layout(a.appMenu()),
		giu.Menu("Device").Layout(a.deviceMenu()),
		giu.Menu("Help").Layout(a.helpMenu()),
	}
}

func (a *App) appMenu() giu.Widget {
	return giu.Layout{
		giu.MenuItem("Calibrate...").OnClick(func() {
			giu.OpenPopup(calibrationPopupID)
		}),
		giu.MenuItem("Firmware update").OnClick(func() {
			giu.OpenPopup(otaUpdatePopupID)
		}),
		giu.Separator(),
		giu.MenuItem("Preferences").OnClick(func() {
			giu.OpenPopup(preferencesPopupID)
		}),
		giu.Separator(),
		giu.MenuItem("Quit").OnClick(a.Quit),
	}
}

func (a *App) deviceMenu() giu.Widget {
	return giu.Layout{
		giu.MenuItem("Set Frequency...").OnClick(func() {
			giu.OpenPopup(frequencyPopupID)
		}).Enabled(a.connected),
	}
}

func (a *App) helpMenu() giu.Widget {
	return giu.Layout{
		giu.MenuItem("GitHub").OnClick(func() {
			browser.OpenURL(GHURL)
			giu.CloseCurrentPopup()
		}),
	}
}

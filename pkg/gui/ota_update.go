package gui

import (
	"github.com/AllenDang/giu"
	"github.com/kpango/glg"
	"github.com/pkg/browser"
	"github.com/sqweek/dialog"
)

const idfURL = "https://docs.espressif.com/projects/esp-idf/en/stable/esp32/get-started/index.html"

func (a *App) otaUpdateDialog() giu.Widget {
	return giu.PopupModal(otaUpdatePopupID).Layout(
		giu.Label(`Over-The-Air Firmware Update for the EspScope device.
Note that you can only use this AFTER the initial flash has been done.

To perform the firmware update follow those steps:
`),
		giu.Bullet(),
		giu.Label("Connect to the EspScope"),
		giu.Condition(
			a.connected, giu.Layout{
				giu.Bullet(),
				giu.Row(
					giu.Label("Take the latest version of the firmware from"),
					giu.Link(GHURL).OnClick(func() {
						browser.OpenURL(GHURL)
					}),
				),
				giu.Bullet(),
				giu.Row(
					giu.Label("Download & install"),
					giu.Link("ESP-IDF").OnClick(func() {
						browser.OpenURL(idfURL)
					}),
				),
				giu.Bullet(),
				giu.Label("Go to EspHome/firmware"),
				giu.Bullet(),
				giu.Label("Execute idf.py build"),
				giu.Bullet(),
				giu.Label("You should get build/EspScope.bin file. Please select it in the field below"),
				giu.Row(
					giu.InputText(&a.otaFile),
					giu.Button("...").OnClick(func() {
						filepath, err := dialog.File().Filter("Firmware file", "bin").SetStartDir(".").Load()
						if err != nil {
							glg.Errorf("Unable to load file path: %v", err)
						}

						a.otaFile = filepath
					}),
				),
				giu.Condition(
					a.otaFile != "",
					giu.Layout{
						giu.Bullet(),
						giu.Row(
							giu.Label("Press the button to start the update:"),
							giu.Button("Start").Disabled(a.otaInProgress).OnClick(func() {
								go func() {
									a.otaInProgress = true
									if err := a.espManager.OTAUpdate(a.otaFile); err != nil {
										glg.Errorf("Error while uploading Over-The-Air update: %v", err)
									}
									a.otaInProgress = false
								}()
							}),
							giu.Custom(func() {
								if !a.otaInProgress {
									return
								}

								const radius = 10.
								giu.ProgressIndicator("loading", 2*radius, 2*radius, radius).Build()
							}),
						),
					},
					giu.Layout{},
				),
			},
			giu.Layout{},
		),
		giu.Separator(),
		giu.Align(giu.AlignRight).To(
			giu.Button("Close").OnClick(giu.CloseCurrentPopup),
		),
	)
}

package gui

import "github.com/AllenDang/giu"

func (a *App) preferencesPopup() giu.Widget {
	return giu.PopupModal(preferencesPopupID).Layout(
		giu.Checkbox("Auto-connect on startup", &a.config.AutoConnect),
		giu.Checkbox("Auto-stream on connect", &a.config.AutoStream),
		giu.Tooltip("Automatically request streaming when connecting to the EspScope server"),
		giu.Row(
			giu.Label("Sample Rate: "),
			giu.InputInt(&a.config.SampleRate).OnChange(func() {
				a.setSampleRate(int(a.config.SampleRate))
			}).Size(sampleRateFieldSize),
		),
		giu.Separator(),
		giu.Align(giu.AlignRight).To(
			giu.Button("Ok").OnClick(func() {
				a.saveConfig()
				giu.CloseCurrentPopup()
			}),
		),
	)
}

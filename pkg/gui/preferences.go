package gui

import "github.com/AllenDang/giu"

func (a *App) preferencesPopup() giu.Widget {
	return giu.PopupModal(preferencesPopupID).Layout(
		giu.Checkbox("Auto-connect on startup", &a.config.AutoConnect),
		giu.Checkbox("Auto-stream on connect", &a.config.AutoStream),
		giu.Tooltip("Automatically request streaming when connecting to the EspScope server"),
		giu.Row(
			giu.Label("Samples per frame: "),
			giu.InputInt(&a.config.SamplesPerFrame).OnChange(func() {
				a.setSamplesPerFrame(int(a.config.SamplesPerFrame))
			}).Size(sampleRateFieldSize),
		),
		giu.Separator(),
		giu.Checkbox("Show current value on plot", &a.config.PosOnCrosshairs),
		giu.Row(
			giu.Label("Line thickness: "),
			giu.SliderInt(&a.config.LineThickness, 1, 10).Size(giu.Auto),
		),
		giu.Row(
			giu.Label("Line color:"),
			giu.ColorEdit("##lineColor", &a.config.LineColor).Size(giu.Auto), // TODO: the placeholder label can be removed: see https://github.com/AllenDang/giu/pull/1077
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

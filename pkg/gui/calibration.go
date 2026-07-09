package gui

import (
	"fmt"

	"github.com/AllenDang/giu"
	"github.com/gucio321/EspScope/pkg/config"
	"github.com/kpango/glg"
)

const (
	calibrationChildW, calibrationChildH = 500, 400
	calibrationPopupID                   = "Calibration##" + config.AppID
)

func (a *App) calibrationPopup() giu.Widget {
	groundCalibration := giu.Child().
		Size(calibrationChildW, calibrationChildH).
		Layout(
			giu.Label(`1st stage: Calibrate GND volatge.
Please short red and black wires, Then press button below.

This is calclated, by taking sampleRate of measurements and calculating the arithmetical average of those samples.
The Drift voltage is then updated by the average value. The drift voltage is used to shift the plot. After this calibration,
you should see the signal laying on the X axis.
`).Wrapped(true),
			giu.Button("Calculate").Size(-1, 0).OnClick(func() {
				avg := a.average
				avg /= a.config.VoltageMultiplyer

				fmt.Println(a.config.DriftVoltage, avg)
				a.config.DriftVoltage += avg

				a.saveConfig()

				glg.Infof("Drift voltage updated by %v. Now is %v", avg, a.config.DriftVoltage)
			}),
		)

	knownVoltageCalibration := giu.Child().
		Size(calibrationChildW, calibrationChildH).
		Layout(
			giu.Label(`Stage 2: Voltage scale calculation.
Take a known voltage (e.g. from lab power supply, or battery checked with multimeter).
Connect scope probes to that known voltage, then, enter the value below and press the button.
`).Wrapped(true),

			giu.Row(
				giu.InputFloat(&a.knownVoltage),
				giu.Button("Calibrate").OnClick(func() {
					avg := a.average

					newVM := float64(a.knownVoltage) / avg

					glg.Infof("Updating voltage multiplyer %v => %v", a.config.VoltageMultiplyer, newVM)

					a.config.VoltageMultiplyer = newVM

					a.saveConfig()
				}),
			),
		)

	return giu.PopupModal(calibrationPopupID).Layout(
		giu.Style().SetFontSize(26).To(
			giu.Label("Calibration"),
		),
		giu.Separator(),
		giu.TabBar().TabItems(
			giu.TabItem("GND Calibration").Layout(groundCalibration),
			giu.TabItem("Voltage Calibration").Layout(knownVoltageCalibration),
		),
		giu.Separator(),
		giu.Align(giu.AlignRight).To(
			giu.Row(
				giu.Button("Close").OnClick(giu.CloseCurrentPopup),
			),
		),
	)
}

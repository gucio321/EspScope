package gui

func (a *App) scopeUpdater() {
	for {
		newData := make([]float64, a.sampleRate)
		for i := range a.sampleRate {
			newData[i] = a.ADCToVoltage(<-a.listener.data)
		}

		if !a.freez {
			copy(a.data, newData)
		}
	}
}

func (a *App) ADCToVoltage(adc uint16) (voltage float64) {
	return (a.lookupTable.ADCToV(adc) - a.config.DriftVoltage) * a.config.VoltageMultiplyer
}

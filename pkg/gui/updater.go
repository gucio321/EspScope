package gui

func (a *App) scopeUpdater() {
	for {
		newData := make([]float64, a.sampleRate)
		avg := float64(0)
		for i := range a.sampleRate {
			newData[i] = a.ADCToVoltage(<-a.listener.data)
			avg += newData[i]
		}

		if !a.freez {
			copy(a.data, newData)
			a.average = avg / float64(a.sampleRate)
		}
	}
}

func (a *App) ADCToVoltage(adc uint16) (voltage float64) {
	return (a.lookupTable.ADCToV(adc) - a.config.DriftVoltage) * a.config.VoltageMultiplyer
}

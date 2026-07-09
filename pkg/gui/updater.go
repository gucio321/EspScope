package gui

func (a *App) scopeUpdater() {
	for {
		newData := make([]float64, a.samplesPerFrame)
		avg := float64(0)
		for i := range a.samplesPerFrame {
			newData[i] = a.ADCToVoltage(<-a.listener.data)
			avg += newData[i]
		}

		if !a.freez {
			copy(a.data, newData)
			a.average = avg / float64(a.samplesPerFrame)
		}
	}
}

func (a *App) ADCToVoltage(adc uint16) (voltage float64) {
	return (a.lookupTable.ADCToV(adc) - a.config.DriftVoltage) * a.config.VoltageMultiplyer
}

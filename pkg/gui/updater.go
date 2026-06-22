package gui

func (a *App) scopeUpdater() {
	for {
		newData := make([]float64, a.sampleRate)
		for i := range a.sampleRate {
			newData[i] = a.ADCToVoltage(float64(<-a.listener.data))
		}

		copy(a.data, newData)
	}
}

func (a *App) ADCToVoltage(adc float64) (voltage float64) {
	// TODO: Those values should be calibrated
	const ADCMax = 4096
	const VMax = 12

	return (adc - ADCMax/2) / ADCMax * VMax
}

package gui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"net"
	"os"
	"sync"
	"time"

	"github.com/AllenDang/cimgui-go/implot"
	"github.com/AllenDang/giu"
	"github.com/dustin/go-humanize"
	"github.com/gucio321/EspScope/pkg/assets"
	"github.com/gucio321/EspScope/pkg/config"
	"github.com/gucio321/EspScope/pkg/gui/espmanager"
	animations "github.com/gucio321/giu-animations/v2"
	"github.com/kpango/glg"
	"golang.org/x/image/colornames"
)

const (
	WindowW, WindowH    = 800, 600
	AppName             = "ESP Scope"
	GHURL               = "https://github.com/gucio321/EspScope"
	IPFieldSize         = 100
	sampleRateFieldSize = 60
	calibrationPopupID  = "Calibration##" + config.AppID
	preferencesPopupID  = "Preferences##" + config.AppID
	ESPSampleRate       = 5e4 // from firmware
	TimeTick            = 1 / ESPSampleRate
	nTicks              = 5
)

var red = color.RGBA{100, 10, 10, 255}

type App struct {
	wnd                   *giu.MasterWindow
	ipStr                 string
	ip                    net.IP
	connected             bool
	listening             bool
	mutex                 *sync.Mutex
	startWelcomeAnimation bool

	listener struct {
		data chan uint16
		end  chan bool
	}

	logo *image.RGBA

	espManager  *espmanager.ESPManager
	lookupTable *espmanager.LookupTable

	sampleRate             int
	xTicks                 []giu.PlotTicker
	data                   []float64
	average                float64
	minVoltage, maxVoltage float64

	config       *config.Config
	knownVoltage float32
	freez        bool

	mouseX  float64
	cursorA Cursor
	cursorB Cursor
}

func New() *App {
	result := &App{
		mutex:       &sync.Mutex{},
		minVoltage:  -12,
		maxVoltage:  12,
		config:      config.NewConfig(),
		lookupTable: espmanager.NewLookupTable(),
	}

	result.setSampleRate(config.DefaultSampleRate)

	result.listener.end = make(chan bool)
	result.listener.data = make(chan uint16, 1024)

	return result
}

func (a *App) Run() (err error) {
	a.startListen()
	defer a.stopListen()

	a.loadConfig()

	go a.scopeUpdater()

	a.wnd = giu.NewMasterWindow(AppName, WindowW, WindowH, 0)

	a.wnd.SetStyle(a.theme())

	a.wnd.SetCloseCallback(a.onClose)

	a.wnd.RegisterKeyboardShortcuts(a.keyboardShortcuts()...)

	a.updatePlotTicks()

	logo, _, err := image.Decode(bytes.NewReader(assets.Logo))
	if err != nil {
		return fmt.Errorf("unable to decode logo image: %v", err)
	}

	a.wnd.SetIcon(logo)

	a.logo = giu.ImageToRgba(logo)

	a.wnd.Run(a.loop)

	return nil
}

func (a *App) loadConfig() {
	glg.Info("Reading config")
	err := a.config.Load()
	switch {
	case errors.Is(err, os.ErrNotExist):
		glg.Debugf("First project run! Welcome! Creating empty config file.")
		a.saveConfig()
	case err != nil:
		glg.Errorf("Loading config file: %s", err)
	}

	a.ipStr = a.config.LastIP
	a.updateIP()

	if a.config.AutoConnect {
		a.connect()
	}
}

func (a *App) loop() {
	animations.Animator(
		animations.Transition(
			func(sf animations.StarterFunc) {
				if !a.startWelcomeAnimation {
					sf.Start(animations.PlayForward)
					a.startWelcomeAnimation = true
				}
				a.welcomeScreen()
			},
			func(sf animations.StarterFunc) {
				a.mainLayout()
			},
		),
	).
		Duration(time.Second).
		Build()
}

func (a *App) welcomeScreen() {
	giu.SingleWindow().Layout(
		giu.Align(giu.AlignCenter).To(
			giu.ImageWithRgba(a.logo),
		),
	)
}

func (a *App) mainLayout() {
	giu.SingleWindowWithMenuBar().Layout(
		a.calibrationPopup(),
		a.preferencesPopup(),
		giu.MenuBar().Layout(a.menuBar()),
		giu.Row(
			giu.Style().
				SetColor(giu.StyleColorFrameBg, func() color.Color {
					switch {
					case a.ip == nil:
						return red
					case a.isConnected():
						return colornames.Green
					default:
						return color.RGBA{}
					}
				}()).SetDisabled(a.isConnected()).
				To(
					giu.InputText(&a.ipStr).OnChange(func() {
						a.updateIP()
					}).Size(IPFieldSize).Hint("192.168.4.1"),
				),
			giu.Style().SetDisabled(a.ip == nil).To(
				giu.Button(func() string {
					if a.isConnected() {
						return "Disconnect"
					}
					return "Connect"
				}()).OnClick(func() {
					if !a.isConnected() == true {
						a.connect()
					} else {
						a.disconnect(true)
					}
				}),
			),
			giu.Style().SetDisabled(!a.isConnected()).To(
				giu.Button("Start stream").OnClick(func() {
					a.startStream()
				}),
				giu.Button("Stop stream").OnClick(func() {
					a.stopStream()
				}),
			),

			giu.Checkbox("Freez", &a.freez),
			giu.Tooltip("Freez the plot. (F)"),

			giu.Custom(func() {
				const indicatorW = 70
				x, _ := giu.GetAvailableRegion()
				is, _ := giu.GetItemSpacing()
				giu.Dummy(x-2*is-indicatorW, 0).Build()
				giu.SameLine()
				giu.Child().Layout(
					giu.Style().SetFontSize(35).To(
						giu.Labelf("%.2f", a.average),
					),
				).Size(indicatorW, 55).Build()
			}),
		),

		giu.Plot("Scope").Plots(
			giu.Line("A0", a.data),
			giu.Custom(func() {
				if implot.IsPlotHovered() {
					mp := implot.GetPlotMousePos()
					a.mouseX = mp.X
				}

				if a.cursorA.Enabled {
					fmt.Println(a.mouseX)
					fmt.Println(a.cursorA)
				}
			}),
		).YLabel("Voltage [V]").
			XFlags(giu.PlotAxisFlagsAutoFit).
			YLimits(a.minVoltage, a.maxVoltage, giu.ConditionOnce).
			Size(-1, -1).
			XTicks(a.xTicks, false).
			XLabel("Time [s]").
			Flags(giu.PlotFlagsCrosshairs),
	)
}

func (a *App) setSampleRate(sr int) {
	a.sampleRate = sr
	a.data = make([]float64, sr)
	a.updatePlotTicks()
}

func (a *App) calibrationPopup() giu.Widget {
	an := animations.Animator(animations.Transition(
		func(sf animations.StarterFunc) {
			giu.Layout{
				giu.Label(`1st stage: Calibrate GND volatge.
Please short read and black wires, Then press button below.

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

					sf.Start(animations.PlayForward)
				}),
			}.Build()
		},

		func(sf animations.StarterFunc) {
			giu.Layout{
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
			}.Build()
		},
	))

	return giu.PopupModal(calibrationPopupID).Layout(
		giu.Style().SetFontSize(26).To(
			giu.Label("Calibration"),
		),
		giu.Separator(),
		giu.Child().Layout(
			an,
		).Size(300, 200),
		giu.Separator(),
		giu.Align(giu.AlignRight).To(
			giu.Row(
				giu.Button("Skip >>").OnClick(func() {
					an.Start(animations.PlayForward)
				}),
				giu.Button("Close").OnClick(giu.CloseCurrentPopup),
			),
		),
	)
}

func (a *App) saveConfig() {
	if err := a.config.Save(); err != nil {
		glg.Errorf("Unable to save config file: %v", err)
	}

	glg.Debugf("Config saved.")
}

func (a *App) updateIP() {
	a.ip = net.ParseIP(a.ipStr)
}

func (a *App) updatePlotTicks() {
	a.xTicks = make([]giu.PlotTicker, nTicks+1)
	for i := range nTicks + 1 {
		a.xTicks[i] = giu.PlotTicker{
			Position: float64(i) * float64(a.sampleRate) / nTicks,
			Label:    humanize.SI(float64(i)*TimeTick*float64(a.sampleRate)/nTicks, "s"),
		}
	}
}

func (a *App) Quit() {
	a.wnd.Close()
}

func (a *App) onClose() bool {
	if a.isConnected() {
		a.stopStream()
	}

	glg.Info("Closing...")

	return true
}

func (a *App) stopStream() {
	if err := a.espManager.StopStream(); err != nil {
		glg.Errorf("Unable to stop stream: %v", err)
	}
}

func (a *App) theme() *giu.StyleSetter {
	result := a.wnd.GetStyle()

	f := giu.Context.FontAtlas.AddFontFromBytes("MonaSans", assets.Font)
	result.SetFont(f)
	result.SetFontSize(24)
	result.
		SetColor(giu.StyleColorWindowBg, colornames.Black).
		SetPlotColor(giu.StylePlotColorFrameBg, colornames.Black).
		SetPlotColor(giu.StylePlotColorPlotBorder, color.RGBA{0, 200, 0, 255}).
		SetPlotColor(giu.StylePlotColorAxisBgHovered, color.RGBA{0, 100, 0, 255}).
		SetColor(giu.StyleColorFrameBg, color.RGBA{0, 60, 0, 255}).
		SetColor(giu.StyleColorButton, color.RGBA{0, 60, 0, 255}).
		SetColor(giu.StyleColorButtonHovered, color.RGBA{0, 120, 0, 255}).
		SetPlotColor(giu.StylePlotColorPlotBg, color.RGBA{0, 20, 0, 255}).
		SetPlotColor(giu.StylePlotColorAxisGrid, color.RGBA{0, 100, 0, 255}).
		SetColor(giu.StyleColorChildBg, color.RGBA{0, 20, 0, 255})

	return result
}

func (a *App) connect() {
	if a.ip == nil {
		glg.Warn("Attempt to connect, but IP empty")
	}

	a.mutex.Lock()
	a.connected = true
	a.mutex.Unlock()

	a.espManager = espmanager.New(a.ip)
	a.config.LastIP = a.ipStr
	a.saveConfig()

	go func() {
		var err error
		a.lookupTable, err = a.espManager.GetLookupTable()
		if err != nil {
			glg.Errorf("Unable to get ADC Lookup Table: %v", err)
			a.disconnect(false)
		}

		if a.config.AutoStream {
			a.startStream()
		}
	}()
}

func (a *App) disconnect(safe bool) {
	if safe {
		if err := a.espManager.EndConnection(); err != nil {
			glg.Warn("Unable to properly end ESP connecction: %v", err)
		}
	}

	a.mutex.Lock()
	a.connected = false
	a.mutex.Unlock()
}

func (a *App) isConnected() bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	return a.connected
}

func (a *App) startStream() {
	if err := a.espManager.StartStream(); err != nil {
		glg.Error("Unable to start stream: %v", err)
	}
}

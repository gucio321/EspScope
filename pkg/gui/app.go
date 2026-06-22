package gui

import (
	"image/color"
	"net"
	"sync"

	"github.com/AllenDang/giu"
	"github.com/gucio321/EspScope/pkg/gui/espmanager"
	"github.com/kpango/glg"
	"golang.org/x/image/colornames"
)

const (
	WindowW, WindowH = 800, 600
	AppName          = "ESP Scope"
	IPFieldSize      = 100
)

var red = color.RGBA{100, 10, 10, 255}

type App struct {
	wnd       *giu.MasterWindow
	ipStr     string
	ip        net.IP
	connected bool
	listening bool
	mutex     *sync.Mutex

	listener struct {
		data chan uint16
		end  chan bool
	}

	espManager *espmanager.ESPManager

	sampleRate int
	data       []float64
}

func New() *App {
	result := &App{
		mutex: &sync.Mutex{},
	}

	result.setSampleRate(1024)

	result.listener.end = make(chan bool)
	result.listener.data = make(chan uint16, 1024)

	return result
}

func (a *App) Run() error {
	a.startListen()
	defer a.stopListen()

	go a.scopeUpdater()

	a.wnd = giu.NewMasterWindow(AppName, WindowW, WindowH, 0)

	a.wnd.Run(a.loop)

	return nil
}

func (a *App) loop() {
	sampleRate := int32(a.sampleRate)
	giu.SingleWindow().Layout(
		giu.Row(
			giu.Style().
				SetColor(giu.StyleColorFrameBg, func() color.Color {
					switch {
					case a.ip == nil:
						return red
					case a.connected:
						return colornames.Green
					default:
						return color.RGBA{}
					}
				}()).SetDisabled(a.connected).
				To(
					giu.InputText(&a.ipStr).OnChange(func() {
						a.ip = net.ParseIP(a.ipStr)
					}).Size(IPFieldSize).Hint("192.168.4.1"),
				),
			giu.Style().SetDisabled(a.ip == nil).To(
				giu.Button(func() string {
					if a.connected {
						return "Disconnect"
					}
					return "Connect"
				}()).OnClick(func() {
					if !a.connected == true {
						a.connected = true
						a.espManager = espmanager.New(a.ip)
					} else {
						if err := a.espManager.EndConnection(); err != nil {
							glg.Warn("Unable to properly end ESP connecction: %v", err)
						}

						a.connected = false
					}
				}),
			),
			giu.Style().SetDisabled(!a.connected).To(
				giu.Button("Start stream").OnClick(func() {
					if err := a.espManager.StartStream(); err != nil {
						glg.Error("Unable to start stream: %v", err)
					}
				}),
				giu.Button("Stop stream").OnClick(func() {
					if err := a.espManager.StopStream(); err != nil {
						glg.Errorf("Unable to stop stream: %v", err)
					}
				}),
			),
			giu.Label("Sample Rate: "),
			giu.InputInt(&sampleRate).OnChange(func() {
				a.setSampleRate(int(sampleRate))
			}),
		),

		giu.Plot("Scope").Plots(
			giu.Line("A0", a.data),
		),
	)
}

func (a *App) setSampleRate(sr int) {
	a.sampleRate = sr
	a.data = make([]float64, sr)
}

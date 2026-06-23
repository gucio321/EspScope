package gui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"net"
	"sync"
	"time"

	"github.com/AllenDang/giu"
	"github.com/gucio321/EspScope/pkg/assets"
	"github.com/gucio321/EspScope/pkg/gui/espmanager"
	animations "github.com/gucio321/giu-animations/v2"
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

	espManager *espmanager.ESPManager

	sampleRate             int
	data                   []float64
	minVoltage, maxVoltage float64
}

func New() *App {
	result := &App{
		mutex:      &sync.Mutex{},
		minVoltage: -12,
		maxVoltage: 12,
	}

	result.setSampleRate(1024)

	result.listener.end = make(chan bool)
	result.listener.data = make(chan uint16, 1024)

	return result
}

func (a *App) Run() (err error) {
	a.startListen()
	defer a.stopListen()

	go a.scopeUpdater()

	a.wnd = giu.NewMasterWindow(AppName, WindowW, WindowH, 0)

	logo, _, err := image.Decode(bytes.NewReader(assets.Logo))
	if err != nil {
		return fmt.Errorf("unable to decode logo image: %v", err)
	}

	a.wnd.SetIcon(logo)

	a.logo = giu.ImageToRgba(logo)

	a.wnd.Run(a.loop)

	return nil
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

		giu.Style().
			SetPlotColor(giu.StylePlotColorPlotBg, color.RGBA{0, 20, 0, 255}).
			SetPlotColor(giu.StylePlotColorAxisGrid, color.RGBA{0, 100, 0, 255}).
			To(
				giu.Plot("Scope").Plots(
					giu.Line("A0", a.data),
				).YLabel("Voltage [V]").
					XFlags(giu.PlotAxisFlagsAutoFit).
					YLimits(a.minVoltage, a.maxVoltage, giu.ConditionOnce).
					Size(-1, -1),
			),
	)
}

func (a *App) setSampleRate(sr int) {
	a.sampleRate = sr
	a.data = make([]float64, sr)
}

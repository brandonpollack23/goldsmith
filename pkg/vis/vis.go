package vis

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// TODO separate out shared and horiz bars.
// TODO set the max number of bars equal to fftData window size.
// TODO make vertical bars
// TODO maybe use phase to determine color or width or something?

// Shared visualizer information.

var defaultKeymap = Keymap{
	quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

type Visualizer interface {
	UpdateVisualizer(newFFTData NewFFTData)
	// You might consider sticking a `Wait(context.Context) error` method on
	// here, and then you wouldn't have to return a wayward channel from the
	// constructors at all.
}

// TODO clean
type GoldsmithModel interface {
	tea.Model
	SetKeymap(k Keymap)
	SetShowFPS(f bool)

	GetCurrentFPS() float64
	GetAverageFPS() float64
	GetFrameCount() int64
}

// nit: could this (and the ShowFPS field) be made private to this package? If
// so I would recommend it, it makes it much easier to understand how some type
// is going to be used in the context of the rest of the program if you know it
// can only be used within this domain.
type GoldsmithSharedFields struct {
	ShowFPS       bool
	startTime     time.Time
	lastFrameTime time.Time
	currentFPS    float64
	frameCount    int64
}

// nit: idiomatically I'd call this `newSharedFields`
func initSharedFields() GoldsmithSharedFields {
	now := time.Now()
	return GoldsmithSharedFields{
		startTime:     now,
		lastFrameTime: now,
	}
}

func (m *GoldsmithSharedFields) SetShowFPS(f bool) {
	m.ShowFPS = f
}

func (m GoldsmithSharedFields) GetAverageFPS() float64 {
	return float64(m.frameCount) / (float64(time.Now().Sub(m.startTime).Seconds()))
}

func (m GoldsmithSharedFields) GetCurrentFPS() float64 {
	return m.currentFPS
}

func (m GoldsmithSharedFields) GetFrameCount() int64 {
	return m.frameCount
}

type Keymap struct {
	quit key.Binding
}

type NewFFTData struct {
	// this is the first time I've ever seen one of the complex types used, well
	// done! :clap:
	Data []complex128
}

func (m *GoldsmithSharedFields) updateFPS() {
	t := time.Now()

	m.frameCount += 1
	frameTime := t.Sub(m.lastFrameTime).Seconds()

	if frameTime != 0 {
		m.currentFPS = (1.0 / frameTime)
	}

	m.lastFrameTime = t
}

func displayFPS(b io.StringWriter, m GoldsmithModel) {
	b.WriteString(fmt.Sprintf("Frame Count: %d\n", m.GetFrameCount()))
	b.WriteString(fmt.Sprintf("Current FPS: %.2f\n", m.GetCurrentFPS()))
	b.WriteString(fmt.Sprintf("Average FPS: %.2f\n", m.GetAverageFPS()))
}

// Shared Options

// I guess this is partially personal preference, but I personally don't like
// these functional options. I'll write a blog post on it one day... but for a
// lot of reasons they end up being inconvenient in the long run. I would just
// pass in an `*Opts` struct. Some people really like them though, so idk maybe
// I'm wrong.
type VisualizerOption func(GoldsmithModel)

func WithKeymap(k Keymap) VisualizerOption {
	return func(v GoldsmithModel) {
		v.SetKeymap(k)
	}
}

func WithFPS(f bool) VisualizerOption {
	return func(v GoldsmithModel) {
		v.SetShowFPS(f)
	}
}

// Launches the bubble tea visualizer and returns the program handle as well as
// a done signal channel.
func launchTeaProgram(m GoldsmithModel, opts []VisualizerOption) (*tea.Program, <-chan struct{}) {
	for _, opt := range opts {
		opt(m)
	}

	p := tea.NewProgram(m, tea.WithoutSignalHandler())

	waitChan := make(chan struct{})
	go func() {
		if _, err := p.Run(); err != nil {
			panic("Error occurred: %s" + err.Error())
		}

		if waitChan != nil {
			// Better to close it, so you could theoretically have multiple
			// things waiting if you wanted. Also the nil check isn't really
			// necessary, it will definitely not be nil.
			waitChan <- struct{}{}
		}
	}()
	return p, waitChan
}

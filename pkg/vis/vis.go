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
}

// TODO clean
type GoldsmithModel interface {
	tea.Model
	SetKeymap(k Keymap)
	SetShowFPS(f bool)

	GetCurrentFPS() float64
	GetAverageFPS() float64
}

type GoldsmithSharedFields struct {
	ShowFPS       bool
	lastFrameTime time.Time
	currentFPS    float64
	averageFPS    float64
	frameCount    int64
}

func initSharedFields() GoldsmithSharedFields {
	return GoldsmithSharedFields{
		lastFrameTime: time.Now(),
	}
}

func (m *GoldsmithSharedFields) SetShowFPS(f bool) {
	m.ShowFPS = f
}

func (m GoldsmithSharedFields) GetAverageFPS() float64 {
	return m.averageFPS
}

func (m GoldsmithSharedFields) GetCurrentFPS() float64 {
	return m.currentFPS
}

type Keymap struct {
	quit key.Binding
}

type NewFFTData struct {
	Data []complex128
}

func (m *GoldsmithSharedFields) updateFPS() {
	t := time.Now()

	lastFrameCount := m.frameCount
	m.frameCount += 1
	frameTime := t.Sub(m.lastFrameTime).Seconds()

	if frameTime != 0 {
		m.currentFPS = (1.0 / frameTime)
		m.averageFPS = ((m.GetAverageFPS() * float64(lastFrameCount)) + m.currentFPS) / float64(m.frameCount)
	}

	m.lastFrameTime = t
}

func displayFPS(b io.StringWriter, m GoldsmithModel) {
	b.WriteString(fmt.Sprintf("Current FPS: %.2f\n", m.GetCurrentFPS()))
	b.WriteString(fmt.Sprintf("Average FPS: %.2f\n", m.GetAverageFPS()))
}

// Shared Options

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
			waitChan <- struct{}{}
		}
	}()
	return p, waitChan
}

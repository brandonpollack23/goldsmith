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

type GoldsmithModel interface {
	tea.Model
	SetKeymap(k Keymap)
	SetShowFPS(f bool)

	getLastFrameTime() time.Time
	setLastFrameTime(t time.Time)
}

type GoldsmithSharedFields struct {
	ShowFPS       bool
	lastFrameTime time.Time
}

func (m *GoldsmithSharedFields) SetShowFPS(f bool) {
	m.ShowFPS = f
}

func (m GoldsmithSharedFields) getLastFrameTime() time.Time {
	return m.lastFrameTime
}

func (m *GoldsmithSharedFields) setLastFrameTime(t time.Time) {
	m.lastFrameTime = t
}

type Keymap struct {
	quit key.Binding
}

type NewFFTData struct {
	Data []complex128
}

func displayFPS(b io.StringWriter, m GoldsmithModel) {
	t := time.Now()
	frameTime := t.Sub(m.getLastFrameTime()).Seconds()
	if t != m.getLastFrameTime() {
		fps := 1.0 / frameTime
		b.WriteString(fmt.Sprintf("FPS: %.2f", fps))
	} else {
		b.WriteString("FPS: inf")
	}
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

package vis

import (
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
}

type Keymap struct {
	quit key.Binding
}

type NewFFTData struct {
	Data []complex128
}

// Shared Options

type VisualizerOption func(GoldsmithModel)

func WithKeymap(k Keymap) VisualizerOption {
	return func(v GoldsmithModel) {
		v.SetKeymap(k)
	}
}

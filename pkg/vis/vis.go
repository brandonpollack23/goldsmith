package vis

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// TODO add exit input
// TODO make horizontal bars instead of a string.

// Shared visualizer information.

var (
	defaultKeymap = Keymap{
		quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
)

type Visualizer interface {
	UpdateVisualizer(newFFTData NewFFTData)
}

type HorizontalBarsVisualizer struct {
	Program *tea.Program
}

func (v HorizontalBarsVisualizer) UpdateVisualizer(newFFTData NewFFTData) {
	v.Program.Send(newFFTData)
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

// HorizontalBarsModel implementation.

type HorizontalBarsModel struct {
	fftData []complex128
	bars    []progress.Model
	keymap  Keymap
}

func NewHorizontalBarsVisualizer(numBars int, opts ...VisualizerOption) *HorizontalBarsVisualizer {
	m := HorizontalBarsModel{
		bars:   make([]progress.Model, numBars),
		keymap: defaultKeymap,
	}

	for _, opt := range opts {
		opt(&m)
	}

	p := tea.NewProgram(m, tea.WithoutSignalHandler())
	go func() {
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error occurred: %s", err.Error())
			os.Exit(1)
		}

		os.Exit(0)
	}()

	return &HorizontalBarsVisualizer{Program: p}
}

func (m HorizontalBarsModel) Init() tea.Cmd {
	return nil
}

func (m HorizontalBarsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case NewFFTData:
		m.fftData = msg.Data
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m HorizontalBarsModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.quit):
		return m, tea.Quit
	}

	return m, nil
}

func (m HorizontalBarsModel) View() string {
	return fmt.Sprintf("%v", m.fftData)
}

func (m HorizontalBarsModel) SetKeymap(k Keymap) {
	m.keymap = k
}

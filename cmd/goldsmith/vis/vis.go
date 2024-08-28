package vis

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// TODO add exit input
// TODO make horizontal bars instead of a string.

type Visualizer interface {
	UpdateVisualizer(newFFTData NewFFTData)
}

type HorizontalBarsVisualizer struct {
	Program *tea.Program
}

func (v HorizontalBarsVisualizer) UpdateVisualizer(newFFTData NewFFTData) {
	v.Program.Send(newFFTData)
}

type HorizontalBarsModel struct {
	fftData []complex128
	bars    []progress.Model
	fps     time.Duration
}

type NewFFTData struct {
	Data []complex128
}

func NewHorizontalBarsVisualizer(numBars int) *HorizontalBarsVisualizer {
	m := HorizontalBarsModel{
		bars: make([]progress.Model, numBars),
	}

	p := tea.NewProgram(m, tea.WithoutSignalHandler())
	go p.Run()

	return &HorizontalBarsVisualizer{Program: p}
}

func (m HorizontalBarsModel) Init() tea.Cmd {
	return nil
}

func (m HorizontalBarsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case NewFFTData:
		m.fftData = msg.Data
	}
	return m, nil
}

func (m HorizontalBarsModel) View() string {
	return fmt.Sprintf("%v", m.fftData)
}

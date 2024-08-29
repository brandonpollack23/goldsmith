package vis

import (
	"fmt"
	"math"
	"math/cmplx"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// TODO separate out bars from shared.
// TODO make vertical bars

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
	fftData      []complex128
	bars         []progress.Model
	maxBarHeight int
	keymap       Keymap
}

func NewHorizontalBarsVisualizer(numBars int, maxBarHeight int, opts ...VisualizerOption) *HorizontalBarsVisualizer {
	bars := make([]progress.Model, numBars)
	for i := range bars {
		bars[i] = progress.New(progress.WithDefaultGradient())
	}

	m := HorizontalBarsModel{
		bars:         bars,
		keymap:       defaultKeymap,
		maxBarHeight: maxBarHeight,
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
		return m, nil

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
	aggregateBars := make([]float64, len(m.bars))

	// TODO maybe use phase to determine color or width or something?
	// TODO set the max number of bars equal to fftData window size.
	// First aggregate bars together from fft window to match requested number of bars.
	// And also convert to logarithmic scale.
	barsToAggregate := len(m.fftData) / len(m.bars)
	for bi := range m.bars {
		for i := range barsToAggregate {
			aggregateBars[bi] += cmplx.Abs(m.fftData[bi*barsToAggregate+i])
		}
		aggregateBars[bi] = math.Log1p(aggregateBars[bi])
		aggregateBars[bi] /= 5
	}

	// var sb strings.Builder
	// fmt.Fprintf(&sb, "Num bars: %d, bars aggregated: %d\n", len(m.bars), barsToAggregate)
	// fmt.Fprintf(&sb, "max %f\n", slices.Max(aggregateBars))
	// for _, value := range aggregateBars {
	// 	fmt.Fprintf(&sb, "%.1f ", value)
	// }

	// TODO remove all the bars, can just use one?
	var sb strings.Builder
	for i, barValue := range aggregateBars {
		fmt.Fprintf(&sb, "%s\n\n", m.bars[i].ViewAs(barValue))
	}

	return sb.String()
}

func (m *HorizontalBarsModel) SetKeymap(k Keymap) {
	m.keymap = k
}

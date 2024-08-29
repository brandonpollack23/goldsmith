package vis

import (
	"fmt"
	"math"
	"math/cmplx"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type HorizontalBarsModel struct {
	fftData      []complex128
	numBars      int
	bar          progress.Model
	maxBarHeight int
	keymap       Keymap
}

func NewHorizontalBarsVisualizer(numBars int, maxBarHeight int, opts ...VisualizerOption) *HorizontalBarsVisualizer {
	bar := progress.New(progress.WithDefaultGradient())

	m := HorizontalBarsModel{
		bar:          bar,
		numBars:      numBars,
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
	aggregateBars := make([]float64, m.numBars)

	// First aggregate bars together from fft window to match requested number of bars.
	// And also convert to logarithmic scale.
	// Divide by 2 to combine the negative frequency components.
	barsToAggregate := len(m.fftData) / m.numBars / 2
	for bi := range m.numBars {
		for i := range barsToAggregate {
			posComponent := cmplx.Abs(m.fftData[bi*barsToAggregate+i])
			negComponent := cmplx.Abs(m.fftData[len(m.fftData)-1-bi*barsToAggregate-i])
			aggregateBars[bi] += posComponent + negComponent
		}
		aggregateBars[bi] = math.Log1p(aggregateBars[bi])
	}
	maxComponent := slices.Max(aggregateBars)
	for i := range m.numBars {
		aggregateBars[i] /= maxComponent
	}

	// TODO remove all the bars, can just use one?
	var sb strings.Builder
	for _, barValue := range aggregateBars {
		fmt.Fprintf(&sb, "%s\n", m.bar.ViewAs(barValue))
	}

	return sb.String()
}

func (m *HorizontalBarsModel) SetKeymap(k Keymap) {
	m.keymap = k
}

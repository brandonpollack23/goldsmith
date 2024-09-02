package vis

import (
	"fmt"
	"math"
	"math/cmplx"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type HorizontalBarsVisualizer struct {
	VisualizerShared
	program *tea.Program
}

func (v HorizontalBarsVisualizer) UpdateVisualizer(newFFTData NewFFTData) {
	v.program.Send(newFFTData)
}

type HorizontalBarsModel struct {
	GoldsmithSharedFields
	fftData      []complex128
	numBars      int
	bar          progress.Model
	maxBarHeight int
}

func NewHorizontalBarsVisualizer(numBars int, maxBarHeight int, opts ...VisualizerOption) *HorizontalBarsVisualizer {
	bar := progress.New(progress.WithDefaultGradient())

	m := HorizontalBarsModel{
		bar:                   bar,
		numBars:               numBars,
		maxBarHeight:          maxBarHeight,
		GoldsmithSharedFields: initSharedFields(defaultKeymap),
	}

	p, doneChan := launchTeaProgram(&m, opts)

	return &HorizontalBarsVisualizer{
		program:          p,
		VisualizerShared: VisualizerShared{done: doneChan},
	}
}

func (m HorizontalBarsModel) Init() tea.Cmd {
	return nil
}

func (m HorizontalBarsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case NewFFTData:
		if msg.Done {
			return m, tea.Quit
		}

		m.updateFPS()
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

	var sb strings.Builder
	for _, barValue := range aggregateBars {
		fmt.Fprintf(&sb, "%s\n", m.bar.ViewAs(barValue))
	}

	if m.showFPS {
		displayFPS(&sb, m.GoldsmithSharedFields)
	}

	return sb.String()
}

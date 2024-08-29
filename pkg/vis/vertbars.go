package vis

import (
	"fmt"
	"math"
	"math/cmplx"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

type VerticalBarsVisualizer struct {
	Program *tea.Program
}

func (v VerticalBarsVisualizer) UpdateVisualizer(newFFTData NewFFTData) {
	v.Program.Send(newFFTData)
}

type VerticalBarsModel struct {
	GoldsmithSharedFields
	fftData []complex128
	numBars int
	// Actual max bar height (as in character height)
	maxBarHeight int
	BarWidth     int
	keymap       Keymap

	TopDown bool

	// TODO color ramp
	Empty      rune
	Full       rune
	FullColor  string
	EmptyColor string
}

func NewVerticalBarsVisualizer(numBars int, maxBarHeight int, opts ...VisualizerOption) *VerticalBarsVisualizer {
	m := VerticalBarsModel{
		numBars:      numBars,
		keymap:       defaultKeymap,
		TopDown:      false,
		maxBarHeight: maxBarHeight,
		BarWidth:     2,
		Full:         '█',
		Empty:        '░',
		FullColor:    "#7571F9",
		EmptyColor:   "#606060",
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

	return &VerticalBarsVisualizer{Program: p}
}

func (m VerticalBarsModel) Init() tea.Cmd {
	return nil
}

func (m VerticalBarsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.lastFrameTime = time.Now()
	switch msg := msg.(type) {
	case NewFFTData:
		m.fftData = msg.Data
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m VerticalBarsModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.quit):
		return m, tea.Quit
	}

	return m, nil
}

func (m VerticalBarsModel) View() string {
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

	return m.verticalBarsView(aggregateBars)
}

func (m *VerticalBarsModel) SetKeymap(k Keymap) {
	m.keymap = k
}

func (m VerticalBarsModel) verticalBarsView(aggregateBarPercents []float64) string {
	padding := " "
	var b strings.Builder

	for i := range m.maxBarHeight {
		row := i
		if !m.TopDown {
			row = m.maxBarHeight - i - 1
		}

		for _, p := range aggregateBarPercents {
			barHeight := int(p * float64(m.maxBarHeight))
			if row < barHeight {
				// Solid fill
				s := termenv.String(string(m.Full)).Foreground(m.color(m.FullColor)).String()
				b.WriteString(strings.Repeat(s, m.BarWidth))
			} else {
				// Empty fill
				e := termenv.String(string(m.Empty)).Foreground(m.color(m.EmptyColor)).String()
				b.WriteString(strings.Repeat(e, m.BarWidth))
			}

			b.WriteString(padding)
		}
		b.WriteRune('\n')
	}

	if m.ShowFPS {
		displayFPS(&b, &m)
	}

	return b.String()
}

func (m VerticalBarsModel) color(c string) termenv.Color {
	return termenv.ColorProfile().Color(c)
}

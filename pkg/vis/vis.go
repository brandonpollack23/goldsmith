package vis

import (
	"context"
	"errors"
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
	Wait(context.Context) error
}

type VisualizerShared struct {
	done <-chan struct{}
}

func (v VisualizerShared) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return errors.New("timeout")
	case <-v.done:
		return nil
	}
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

type GoldsmithSharedFields struct {
	ShowFPS       bool
	startTime     time.Time
	lastFrameTime time.Time
	currentFPS    float64
	frameCount    int64
}

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
	return float64(m.frameCount) / (float64(time.Since(m.startTime).Seconds()))
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
	Data []complex128
	Done bool
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

func displayFPS(b io.StringWriter, m GoldsmithModel) error {
	_, err := b.WriteString(fmt.Sprintf("Frame Count: %d\n", m.GetFrameCount()))
	if err != nil {
		return err
	}

	_, err = b.WriteString(fmt.Sprintf("Current FPS: %.2f\n", m.GetCurrentFPS()))
	if err != nil {
		return err
	}

	_, err = b.WriteString(fmt.Sprintf("Average FPS: %.2f\n", m.GetAverageFPS()))
	if err != nil {
		return err
	}

	return nil
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

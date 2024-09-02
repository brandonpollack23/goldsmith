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
	done <-chan error
}

func (v VisualizerShared) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return errors.New("timeout")
	case err := <-v.done:
		return err
	}
}

// TODO clean
type GoldsmithModel interface {
	tea.Model
	SetKeymap(k Keymap)
	SetShowFPS(f bool)
}

type GoldsmithSharedFields struct {
	showFPS bool
	keymap  Keymap

	startTime     time.Time
	lastFrameTime time.Time
	currentFPS    float64
	frameCount    int64
}

func initSharedFields(keymap Keymap) GoldsmithSharedFields {
	now := time.Now()
	return GoldsmithSharedFields{
		keymap:        keymap,
		startTime:     now,
		lastFrameTime: now,
	}
}

func (m *GoldsmithSharedFields) SetShowFPS(f bool) {
	m.showFPS = f
}

func (m *GoldsmithSharedFields) SetKeymap(k Keymap) {
	m.keymap = k
}

func (m GoldsmithSharedFields) AverageFPS() float64 {
	return float64(m.frameCount) / (float64(time.Since(m.startTime).Seconds()))
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

func displayFPS(b io.StringWriter, m GoldsmithSharedFields) error {
	_, err := b.WriteString(fmt.Sprintf("Frame Count: %d\n", m.frameCount))
	if err != nil {
		return err
	}

	_, err = b.WriteString(fmt.Sprintf("Current FPS: %.2f\n", m.currentFPS))
	if err != nil {
		return err
	}

	_, err = b.WriteString(fmt.Sprintf("Average FPS: %.2f\n", m.AverageFPS()))
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
func launchTeaProgram(m GoldsmithModel, opts []VisualizerOption) (*tea.Program, <-chan error) {
	for _, opt := range opts {
		opt(m)
	}

	p := tea.NewProgram(m, tea.WithoutSignalHandler())

	waitChan := make(chan error)
	go func() {
		if _, err := p.Run(); err != nil {
			waitChan <- err
			return
		}

		waitChan <- nil
	}()

	return p, waitChan
}

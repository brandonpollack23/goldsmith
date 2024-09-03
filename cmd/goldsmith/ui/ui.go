package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/brandonpollack23/goldsmith/pkg/fft"
	"github.com/brandonpollack23/goldsmith/pkg/vis"
	"go.opentelemetry.io/otel"
)

const name = "github.com/brandonpollack23/goldsmith/cmd/goldsmith/ui"

var (
	tracer = otel.Tracer(name)
)

type UiKeyType int

const (
	FFTDeadlineKey UiKeyType = iota
)

// FFTStreamer buffers a streamer and also computes an FFT whos chunks are available on [FFTChan].
type FFTStreamer interface {
	// Returns the next FFT window when it is time to update the UI.
	NextFFTWindow(context.Context) (fft.FFTWindow, bool, error)
}

type UpdateLoopHandle struct {
	errChan <-chan error
}

func (h UpdateLoopHandle) Wait() error {
	return <-h.errChan
}

func UpdateLoop(ctx context.Context, s FFTStreamer, visualizer vis.Visualizer) error {
	exitChan := make(chan error)
	go func() {
		err := visualizer.Wait(ctx)
		if err != nil {
			exitChan <- fmt.Errorf("visualizer somehow running longer than audio file: %w", err)
			return
		}
		exitChan <- nil
	}()

	for {
		ctx, trace := tracer.Start(ctx, "updateLoop.iteration")

		select {
		case err := <-exitChan:
			return err
		default:
			fftCtx, cancel := context.WithDeadline(ctx,
				time.Now().Add(ctx.Value(FFTDeadlineKey).(time.Duration)))
			nextFFTWindow, ok, err := s.NextFFTWindow(fftCtx)
			cancel()

			if err != nil {
				return err
			}

			visualizer.UpdateVisualizer(vis.NewFFTData{Data: nextFFTWindow.Data, Done: !ok})
		}

		trace.End()
	}
}

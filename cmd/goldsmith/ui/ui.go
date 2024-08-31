package ui

import (
	"context"
	"time"

	"github.com/brandonpollack23/goldsmith/pkg/fft"
	"github.com/brandonpollack23/goldsmith/pkg/vis"
)

const (
	FFTDeadlineKey = iota
)

// FFTStreamer buffers a streamer and also computes an FFT whos chunks are available on [FFTChan].
type FFTStreamer interface {
	// Returns the next FFT window when it is time to update the UI.
	NextFFTWindow(context.Context) (fft.FFTWindow, error)
}

func UIUpdateLoop(ctx context.Context, s FFTStreamer, visualizer vis.Visualizer, exitChan <-chan struct{}) error {
	for {
		select {
		case <-exitChan:
			return nil
		default:
			fftCtx, cancel := context.WithDeadline(ctx, time.Now().Add(ctx.Value(FFTDeadlineKey).(time.Duration)))
			nextFFTWindow, err := s.NextFFTWindow(fftCtx)
			if err != nil {
				return err
			}
			cancel()

			visualizer.UpdateVisualizer(vis.NewFFTData{Data: nextFFTWindow.Data})
		}
	}
}

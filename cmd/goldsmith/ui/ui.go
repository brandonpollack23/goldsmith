package ui

import (
	"context"
	"fmt"
	"log"
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
	NextFFTWindow(context.Context) (fft.FFTWindow, bool, error)
}

func UIUpdateLoop(ctx context.Context, s FFTStreamer, visualizer vis.Visualizer) error {
	exitChan := make(chan struct{})
	go func() {
		err := visualizer.Wait(ctx)
		if err != nil {
			log.Println(fmt.Errorf("visualizer somehow running longer than audio file: %w", err).Error())
		}
		exitChan <- struct{}{}
	}()

	for {
		select {
		case <-exitChan:
			return nil
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
	}
}

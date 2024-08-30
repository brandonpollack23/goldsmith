package ui

import (
	"github.com/brandonpollack23/goldsmith/pkg/fft"
	"github.com/brandonpollack23/goldsmith/pkg/vis"
	"github.com/gopxl/beep"
)

// FFTStreamer buffers a streamer and also computes an FFT whos chunks are available on [FFTChan].
type FFTStreamer interface {
	beep.Streamer
	// Synchronization signal to update FFT to display.
	FFTUpdateSignal() <-chan struct{}
	// Channel that contains FFT window data.
	FFTChan() <-chan fft.FFTWindow
}

func UIUpdateLoop(s FFTStreamer, visualizer vis.Visualizer) {
	for range s.FFTUpdateSignal() {
		fftWindow := <-s.FFTChan()
		visualizer.UpdateVisualizer(vis.NewFFTData{Data: fftWindow.Data})
	}
}

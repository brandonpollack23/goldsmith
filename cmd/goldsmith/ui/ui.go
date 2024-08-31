package ui

import (
	"github.com/brandonpollack23/goldsmith/pkg/fft"
	"github.com/brandonpollack23/goldsmith/pkg/vis"
	"github.com/gopxl/beep"
)

// FFTStreamer buffers a streamer and also computes an FFT whos chunks are available on [FFTChan].
type FFTStreamer interface {
	beep.Streamer // Are the methods from this being used? if not I'd leave it out

	// 1) Could FFTUpdateSignal and FFTChan be consolidated into a single
	// channel? It seems to me like FFTUpdateSignal is redundant.

	// 2) I would also recommend implementing "Next" style methods, rather than
	// returning channels across interface boundaries. E.g.
	//
	//	// Context is used to unblock, method returns `ctx.Err()` in that case
	//	NextFFTWindow(context.Context) (fft.FFTWindow, error)
	//
	// By doing this you're making it significantly easier to mock the interface
	// if you want, and you also hide implementation details of the interface.
	// The implementation may well just be doing a select within the Next
	// method, but it also might have a different way of implementing. In a lot
	// of cases you'll find you can eliminate go-routines from the
	// implementation by designing this way.
	//
	// Another benefit is that it's much easier to reason about how to use the
	// interface. If FFTStreamer were to have a "shutdown" style method, what
	// happens to the channels in that case? Do they need to be blocked
	// on/drained after shutdown? Will they be closed? If the channels are
	// buffered then are the buffered values still valid after shutdown? The
	// answers to these need to be documented and followed, or you risk getting
	// deadlocks or making it impossible to cleanly shutdown. But using a
	// "Next"-style method makes it much easier to reason about the shutdown
	// case: just don't call Shutdown at the same time that something is blocked
	// on Next, and you're good.

	// Synchronization signal to update FFT to display.
	FFTUpdateSignal() <-chan struct{}
	// Channel that contains FFT window data.
	FFTChan() <-chan fft.FFTWindow
}

func UIUpdateLoop(s FFTStreamer, visualizer vis.Visualizer, waitChan <-chan struct{}) {
	for {
		select {
		case <-s.FFTUpdateSignal():
			fftWindow := <-s.FFTChan()
			visualizer.UpdateVisualizer(vis.NewFFTData{Data: fftWindow.Data})
		case <-waitChan:
			return
		}
	}
}

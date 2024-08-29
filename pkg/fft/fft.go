package fft

import (
	"github.com/gopxl/beep"
	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
)

const (
	bufferSizes = 10
)

// FFTStreamer buffers a streamer and also computes an FFT whos chunks are available on [FFTChan].
type FFTStreamer interface {
	beep.Streamer
	// Synchronization signal to update FFT to display.
	FFTUpdateSignal() <-chan struct{}
	// Channel that contains FFT window data.
	FFTChan() <-chan FFTWindow
}

type FFTStreamerImpl struct {
	s                    beep.Streamer
	fftWindowSize        uint32
	fftWindowBuffer      [][2]float64
	fftWindowBufferStart int

	fftInputChan  chan [][2]float64
	fftWindowChan <-chan FFTWindow

	// Synchronization signal to update FFT to display.
	FFTUpdateSignalChan  chan struct{}
	bytesSinceLastWindow uint32
}

func NewFFTStreamer(streamer beep.Streamer, fftWindowSize int, format beep.Format) FFTStreamerImpl {
	internalBufferSize := fftWindowSize * bufferSizes

	fftInputChan := make(chan [][2]float64, bufferSizes)
	fftOutputChan := make(chan FFTWindow, bufferSizes)
	go doFFTs(fftInputChan, fftOutputChan, fftWindowSize)

	return FFTStreamerImpl{
		s:                    streamer,
		fftWindowSize:        uint32(fftWindowSize),
		fftWindowBuffer:      make([][2]float64, internalBufferSize),
		fftWindowBufferStart: internalBufferSize,

		fftInputChan:        fftInputChan,
		fftWindowChan:       fftOutputChan,
		FFTUpdateSignalChan: make(chan struct{}, bufferSizes),
	}
}

func (f *FFTStreamerImpl) Stream(samples [][2]float64) (int, bool) {
	copiedFromLastRead := copy(samples, f.fftWindowBuffer[f.fftWindowBufferStart:])
	checkFFTSyncSignal(f, copiedFromLastRead)

	if copiedFromLastRead == len(samples) {
		f.fftWindowBufferStart += copiedFromLastRead
		return copiedFromLastRead, true
	}

	_, ok := f.s.Stream(f.fftWindowBuffer)
	copiedThisRead := copy(samples[copiedFromLastRead:], f.fftWindowBuffer)
	f.fftWindowBufferStart = copiedThisRead
	checkFFTSyncSignal(f, copiedThisRead)

	fftCopy := make([][2]float64, len(f.fftWindowBuffer))
	copy(fftCopy, f.fftWindowBuffer)
	f.fftInputChan <- fftCopy

	if !ok {
		close(f.fftInputChan)
		close(f.FFTUpdateSignalChan)
	}

	return copiedFromLastRead + copiedThisRead, ok
}

func checkFFTSyncSignal(f *FFTStreamerImpl, bytesCopied int) {
	f.bytesSinceLastWindow += uint32(bytesCopied)
	if f.bytesSinceLastWindow >= f.fftWindowSize {
		f.FFTUpdateSignalChan <- struct{}{}
		f.bytesSinceLastWindow -= f.fftWindowSize
	}
}

func (f *FFTStreamerImpl) Err() error {
	return nil
}

func (f *FFTStreamerImpl) FFTChan() <-chan FFTWindow {
	return f.fftWindowChan
}

func (f *FFTStreamerImpl) FFTUpdateSignal() <-chan struct{} {
	return f.FFTUpdateSignalChan
}

type FFTWindow struct {
	Data []complex128
}

func doFFTs(fftInputChan chan [][2]float64, fftOutputChan chan FFTWindow, fftWindowSize int) {
	for inChunk := range fftInputChan {
		for _, in := range splitSlices(inChunk, fftWindowSize) {
			timeDomain := toMono(in)
			window.Apply(timeDomain, window.Hann)
			freqDomain := fft.FFTReal(timeDomain)

			fftOutputChan <- FFTWindow{
				Data: freqDomain,
			}
		}
	}

	close(fftOutputChan)
}

func splitSlices[T any](s []T, size int) [][]T {
	var result [][]T
	for i := 0; i < len(s); i += size {
		expectedEnd := i + size
		end := min(expectedEnd, len(s))

		result = append(result, s[i:end])
	}

	return result
}

func toMono(x [][2]float64) []float64 {
	result := make([]float64, len(x))
	for i := range len(x) {
		result[i] = (x[i][0] + x[i][1]) / 2
	}

	return result
}

package fft

import (
	"context"
	"errors"

	"github.com/gopxl/beep"
	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
)

const (
	bufferSizes = 10
)

type FFTStreamerImpl struct {
	s                    beep.Streamer
	fftWindowSize        uint32
	fftWindowBuffer      [][2]float64
	fftWindowBufferStart uint32

	fftInputChan  chan [][2]float64
	fftWindowChan <-chan FFTWindow

	// Synchronization signal to update FFT to display.
	fftUpdateSignalChan  chan struct{}
	bytesSinceLastWindow uint32
}

func NewFFTStreamer(streamer beep.Streamer, fftWindowSize uint32, format beep.Format) FFTStreamerImpl {
	internalBufferSize := fftWindowSize * bufferSizes

	fftInputChan := make(chan [][2]float64, bufferSizes)
	fftOutputChan := make(chan FFTWindow, bufferSizes)
	go doFFTs(fftInputChan, fftOutputChan, fftWindowSize)

	return FFTStreamerImpl{
		s:                    streamer,
		fftWindowSize:        fftWindowSize,
		fftWindowBuffer:      make([][2]float64, internalBufferSize),
		fftWindowBufferStart: internalBufferSize,

		fftInputChan:        fftInputChan,
		fftWindowChan:       fftOutputChan,
		fftUpdateSignalChan: make(chan struct{}, bufferSizes),
	}
}

func (f *FFTStreamerImpl) NextFFTWindow(ctx context.Context) (FFTWindow, error) {
	select {
	case <-f.fftUpdateSignalChan:
		return <-f.fftWindowChan, nil
	case <-ctx.Done():
		return FFTWindow{}, errors.New("fft streamer canceled")
	}
}

func (f FFTStreamerImpl) Err() error {
	return f.s.Err()
}

func (f *FFTStreamerImpl) Stream(samples [][2]float64) (int, bool) {
	copiedFromLastRead := copy(samples, f.fftWindowBuffer[f.fftWindowBufferStart:])
	checkFFTSyncSignal(f, copiedFromLastRead)

	if copiedFromLastRead == len(samples) {
		f.fftWindowBufferStart += uint32(copiedFromLastRead)
		return copiedFromLastRead, true
	}

	_, ok := f.s.Stream(f.fftWindowBuffer)
	copiedThisRead := copy(samples[copiedFromLastRead:], f.fftWindowBuffer)
	f.fftWindowBufferStart = uint32(copiedThisRead)
	checkFFTSyncSignal(f, copiedThisRead)

	fftCopy := make([][2]float64, len(f.fftWindowBuffer))
	copy(fftCopy, f.fftWindowBuffer)
	f.fftInputChan <- fftCopy

	if !ok {
		close(f.fftInputChan)
		close(f.fftUpdateSignalChan)
	}

	return copiedFromLastRead + copiedThisRead, ok
}

func checkFFTSyncSignal(f *FFTStreamerImpl, bytesCopied int) {
	f.bytesSinceLastWindow += uint32(bytesCopied)
	if f.bytesSinceLastWindow >= f.fftWindowSize {
		f.fftUpdateSignalChan <- struct{}{}
		f.bytesSinceLastWindow -= f.fftWindowSize
	}
}

type FFTWindow struct {
	Data []complex128
}

func doFFTs(fftInputChan chan [][2]float64, fftOutputChan chan FFTWindow, fftWindowSize uint32) {
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

func splitSlices[T any](s []T, size uint32) [][]T {
	var result [][]T
	for i := 0; i < len(s); i += int(size) {
		expectedEnd := i + int(size)
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

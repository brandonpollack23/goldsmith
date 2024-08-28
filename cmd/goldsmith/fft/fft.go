package fft

import (
	"github.com/gopxl/beep"
	"github.com/mjibson/go-dsp/fft"
)

const (
	bufferSizes = 5
)

type FFTStreamer interface {
	beep.Streamer
	FFTChan() <-chan FFTWindow
}

type fftstreamer struct {
	s                    beep.Streamer
	fftWindowSize        uint32
	fftWindowBuffer      [][2]float64
	fftWindowBufferStart int

	fftInputChan  chan [][2]float64
	fftWindowChan <-chan FFTWindow
}

func NewFFTStreamer(streamer beep.Streamer, fftWindowSize int, format beep.Format) fftstreamer {
	internalBufferSize := fftWindowSize * bufferSizes

	fftInputChan := make(chan [][2]float64, bufferSizes)
	fftOutputChan := make(chan FFTWindow, bufferSizes)
	go doFFTs(fftInputChan, fftOutputChan, fftWindowSize)

	return fftstreamer{
		s:                    streamer,
		fftWindowSize:        uint32(fftWindowSize),
		fftWindowBuffer:      make([][2]float64, internalBufferSize),
		fftWindowBufferStart: internalBufferSize,

		fftInputChan:  fftInputChan,
		fftWindowChan: fftOutputChan,
	}
}

func (f *fftstreamer) Stream(samples [][2]float64) (int, bool) {
	copiedFromLastRead := copy(samples, f.fftWindowBuffer[f.fftWindowBufferStart:])
	if copiedFromLastRead == len(samples) {
		f.fftWindowBufferStart += copiedFromLastRead
		return copiedFromLastRead, true
	}

	_, ok := f.s.Stream(f.fftWindowBuffer)
	copiedThisRead := copy(samples[copiedFromLastRead:], f.fftWindowBuffer)
	f.fftWindowBufferStart = copiedThisRead

	fftCopy := make([][2]float64, len(f.fftWindowBuffer))
	copy(fftCopy, f.fftWindowBuffer)
	f.fftInputChan <- fftCopy

	if !ok {
		close(f.fftInputChan)
	}

	return copiedFromLastRead + copiedThisRead, ok
}

func (f *fftstreamer) Err() error {
	return nil
}

func (f *fftstreamer) FFTChan() <-chan FFTWindow {
	return f.fftWindowChan
}

type FFTWindow struct {
	Data []complex128
}

func doFFTs(fftInputChan chan [][2]float64, fftOutputChan chan FFTWindow, fftWindowSize int) {
	for inChunk := range fftInputChan {
		for _, in := range splitSlices(inChunk, fftWindowSize) {
			timeDomain := toMono(in)
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

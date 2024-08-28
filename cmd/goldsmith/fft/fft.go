package fft

import (
	"time"

	"github.com/gopxl/beep"
	"github.com/mjibson/go-dsp/fft"
)

type FFTStreamer struct {
	s                    beep.Streamer
	fftWindowSize        uint32
	fftWindowBuffer      [][2]float64
	fftWindowBufferStart int

	fftInputChan  chan [][2]float64
	FFTWindowChan <-chan FFTWindow
}

func NewFFTStreamer(streamer beep.Streamer, targetFPS uint32, format beep.Format) FFTStreamer {
	fftWindowSize := format.SampleRate.N(time.Duration(float64(time.Second) / float64(targetFPS)))

	internalBufferSize := fftWindowSize * 2

	fftInputChan := make(chan [][2]float64, 2)
	fftOutputChan := make(chan FFTWindow, 2)
	go doFFTs(fftInputChan, fftOutputChan, fftWindowSize)

	return FFTStreamer{
		s:                    streamer,
		fftWindowSize:        uint32(fftWindowSize),
		fftWindowBuffer:      make([][2]float64, internalBufferSize),
		fftWindowBufferStart: internalBufferSize,

		fftInputChan:  fftInputChan,
		FFTWindowChan: fftOutputChan,
	}
}

func (f *FFTStreamer) Stream(samples [][2]float64) (int, bool) {
	copiedFromLastRead := copy(samples, f.fftWindowBuffer[f.fftWindowBufferStart:])
	if copiedFromLastRead == len(samples) {
		f.fftWindowBufferStart += copiedFromLastRead
		return copiedFromLastRead, true
	}

	_, ok := f.s.Stream(f.fftWindowBuffer)
	copiedThisRead := copy(samples[copiedFromLastRead:], f.fftWindowBuffer)
	f.fftWindowBufferStart = copiedThisRead

	var fftCopy [][2]float64
	copy(fftCopy, f.fftWindowBuffer)
	f.fftInputChan <- fftCopy

	if !ok {
		close(f.fftInputChan)
	}

	return copiedFromLastRead + copiedThisRead, ok
}

func (f *FFTStreamer) Err() error {
	return nil
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

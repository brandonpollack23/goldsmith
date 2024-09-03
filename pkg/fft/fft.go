package fft

import (
	"context"
	"errors"

	"github.com/gopxl/beep"
	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const name = "github.com/brandonpollack23/goldsmith/pkg/fft"

var (
	tracer = otel.Tracer(name)
	meter  = otel.Meter(name)
)

const (
	bufferSizes = 10
)

type FFTStreamerImpl struct {
	ctx context.Context
	s   beep.Streamer

	fftWindowSize        uint32
	fftWindowBuffer      [][2]float64
	fftWindowBufferStart uint32

	doFFTDone     <-chan error
	fftInputChan  chan [][2]float64
	fftWindowChan <-chan FFTWindow

	// Synchronization signal to update FFT to display.
	fftUpdateSignalChan  chan struct{}
	bytesSinceLastWindow uint32
}

func NewFFTStreamer(
	ctx context.Context,
	streamer beep.Streamer,
	fftWindowSize uint32,
	format beep.Format,
) FFTStreamerImpl {
	internalBufferSize := fftWindowSize * bufferSizes

	fftInputChan := make(chan [][2]float64, bufferSizes)
	fftOutputChan := make(chan FFTWindow, bufferSizes)

	doFFTDone := make(chan error)
	go func() {
		err := doFFTs(ctx, fftInputChan, fftOutputChan, fftWindowSize)
		doFFTDone <- err
		close(fftOutputChan)
	}()

	return FFTStreamerImpl{
		ctx:                  ctx,
		s:                    streamer,
		fftWindowSize:        fftWindowSize,
		fftWindowBuffer:      make([][2]float64, internalBufferSize),
		fftWindowBufferStart: internalBufferSize,

		doFFTDone:           doFFTDone,
		fftInputChan:        fftInputChan,
		fftWindowChan:       fftOutputChan,
		fftUpdateSignalChan: make(chan struct{}, bufferSizes),
	}
}

func (f *FFTStreamerImpl) NextFFTWindow(ctx context.Context) (FFTWindow, bool, error) {
	ctx, span := tracer.Start(ctx, "NextFFTWindow")
	defer span.End()

	select {
	case _, ok := <-f.fftUpdateSignalChan:
		if !ok {
			return FFTWindow{}, false, nil
		}
		return <-f.fftWindowChan, true, nil
	case <-ctx.Done():
		return FFTWindow{}, false, errors.New("fft streamer canceled")
	}
}

func (f FFTStreamerImpl) Err() error {
	sErr := f.s.Err()
	if sErr != nil {
		return sErr
	}

	select {
	case err := <-f.doFFTDone:
		return err
	default:
		return nil
	}
}

func (f *FFTStreamerImpl) Stream(samples [][2]float64) (int, bool) {
	ctx, span := tracer.Start(f.ctx, "FFTStreamer.Stream")
	defer span.End()

	ctx, span = tracer.Start(ctx, "FFTStreamer.Stream.buffer")
	copiedFromLastRead := copy(samples, f.fftWindowBuffer[f.fftWindowBufferStart:])
	checkFFTSyncSignal(f, copiedFromLastRead)
	span.End()

	if copiedFromLastRead == len(samples) {
		f.fftWindowBufferStart += uint32(copiedFromLastRead)
		return copiedFromLastRead, true
	}

	ctx, span = tracer.Start(ctx, "FFTStreamer.Stream.underlying")
	_, ok := f.s.Stream(f.fftWindowBuffer)
	span.End()

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

func doFFTs(ctx context.Context, fftInputChan chan [][2]float64, fftOutputChan chan FFTWindow, fftWindowSize uint32) error {
	ctx, span := tracer.Start(ctx, "FFT Manager")
	defer span.End()

	fftCount, err := meter.Int64Counter("fft.count", metric.WithDescription("number of FFT windows calcualted"))
	if err != nil {
		return err
	}

	for inChunk := range fftInputChan {
		splits := splitSlices(inChunk, fftWindowSize)
		ctx, span := tracer.Start(
			ctx,
			"FFT Chunk",
			trace.WithAttributes(attribute.KeyValue{
				Key:   "numSlices",
				Value: attribute.IntValue(len(splits)),
			}),
		)

		for _, in := range splits {
			ctx, span := tracer.Start(ctx, "fft")

			timeDomain := toMono(in)
			window.Apply(timeDomain, window.Hann)
			freqDomain := fft.FFTReal(timeDomain)

			fftCount.Add(ctx, 1)
			fftOutputChan <- FFTWindow{
				Data: freqDomain,
			}

			span.End()
		}

		span.End()
	}

	return nil
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

package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"time"

	"github.com/brandonpollack23/goldsmith/cmd/goldsmith/ui"
	"github.com/brandonpollack23/goldsmith/pkg/fft"
	"github.com/brandonpollack23/goldsmith/pkg/vis"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/wav"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// TODO display playback bar at the bottom with timestamp and max time etc.
// TODO Volume with beep
// TODO other beep effects?
// TODO animations on bars using harmonica (like progress has)?
// TODO add CTRL for play/pause

var (
	targetFPS  uint32
	visType    string
	showFPS    bool
	cpuProfile string
	memProfile string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "goldsmith [music filename]",
		Short: "A cli based music visualizer written in go",
		Long: `This is a cli application built on bubbletea/bubbles and some go fft libraries 
and audio libraries to bring you some magic bars for visualization. Maybe one day a gui etc too.`,
		Args: cobra.ExactArgs(1),
		// Uncomment the following line if your bare application
		// has an action associated with it:
		RunE: runVisualizer,
	}

	rootCmd.PersistentFlags().Uint32VarP(&targetFPS, "target_fps", "f", 30,
		"The updates FPS for the visualizer, affects FFT window")
	rootCmd.PersistentFlags().StringVarP(&visType, "visualizer", "v", "vertical_bars",
		"Which visualizer type to use")
	rootCmd.PersistentFlags().BoolVarP(&showFPS, "showfps", "s", false,
		"Show FPS below visualizer")

	err := rootCmd.RegisterFlagCompletionFunc("vertical_bars", func(cmd *cobra.Command, args []string,
		toComplete string,
	) ([]string, cobra.ShellCompDirective) {
		return []string{"horizontal_bars", "vertical_bars"}, cobra.ShellCompDirectiveNoFileComp
	})
	if err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().StringVarP(&cpuProfile, "cpu_profile", "p", "",
		"Emit CPU and memory profiles and an execution trace to '[filename].[pid].{cpu,trace}', respectively")
	rootCmd.PersistentFlags().StringVarP(&memProfile, "mem_profile", "m", "",
		"Emit CPU and memory profiles and an execution trace to '[filename].[pid].mem', respectively")

	err = rootCmd.Execute()
	if err != nil {
		panic("Fatal error: " + err.Error())
	}
}

func runVisualizer(cmd *cobra.Command, args []string) error {
	if cpuProfile != "" {
		var (
			stop func()
			err  error
		)
		if stop, err = initCPUProfiling(cpuProfile, 0); err != nil {
			return err
		}
		defer stop()
	}

	if memProfile != "" {
		defer saveMemoryProfile(memProfile)
	}

	audioFile, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer audioFile.Close()

	streamer, format, err := decodeAudioFile(audioFile)
	if err != nil {
		return fmt.Errorf("error decoding file %s: %w", audioFile.Name(), err)
	}
	defer streamer.Close()

	windowDuration := time.Duration(float64(time.Second) / float64(targetFPS))
	fftWindowSize := uint32(format.SampleRate.N(windowDuration))
	fftStreamer := fft.NewFFTStreamer(streamer, fftWindowSize, format)
	songDuration := format.SampleRate.D(streamer.Len())

	ctx := context.Background()
	ctx = context.WithValue(ctx, ui.FFTDeadlineKey, 2*windowDuration)
	ctx, cancel := context.WithTimeout(ctx, songDuration+2*time.Second)
	defer cancel()

	// Initialize the speaker to use the sample rate of the audio file selected.
	// I can also use beep.Resample around the streamer to always use a specific
	// output sample rate for everything no matter the input.
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		return fmt.Errorf("cannot initializer speaker: %w", err)
	}

	speaker.Play(&fftStreamer)
	var visualizer vis.Visualizer
	switch visType {
	case "horizontal_bars":
		visualizer = vis.NewHorizontalBarsVisualizer(32,
			int(math.Pow(2, float64(8*format.Precision))), vis.WithFPS(showFPS))
	case "vertical_bars":
		visualizer = vis.NewVerticalBarsVisualizer(64, 40, vis.WithFPS(showFPS))
	default:
		panic("unknown visualizer type: " + visType)
	}

	err = ui.UIUpdateLoop(ctx, &fftStreamer, visualizer)
	if err != nil {
		return err
	}

	return nil
}

func decodeAudioFile(audioFile *os.File) (beep.StreamSeekCloser, beep.Format, error) {
	var streamer beep.StreamSeekCloser
	var format beep.Format
	var err error

	extension := filepath.Ext(audioFile.Name())
	switch extension {
	case ".mp3":
		streamer, format, err = mp3.Decode(audioFile)
	case ".wav":
		streamer, format, err = wav.Decode(audioFile)
		// TODO other formats (flac, vorbis, midi, etc)
	default:
		return nil, beep.Format{}, fmt.Errorf("unsupported audio file format: %v", extension)
	}

	return streamer, format, err
}

// Initializes profiling and returns a function to defer to stop it.
func initCPUProfiling(prefix string, memProfileRate int) (func(), error) {
	cpu, err := os.Create(fmt.Sprintf("%s.%v.cpu", prefix, os.Getpid()))
	if err != nil {
		return nil, errors.Wrap(err, "could not start CPU profile")
	}
	if err = pprof.StartCPUProfile(cpu); err != nil {
		return nil, errors.Wrap(err, "could not start CPU profile")
	}

	exec, err := os.Create(fmt.Sprintf("%s.%v.trace", prefix, os.Getpid()))
	if err != nil {
		return nil, errors.Wrap(err, "could not start execution trace")
	}
	if err = trace.Start(exec); err != nil {
		return nil, errors.Wrap(err, "could not start execution trace")
	}

	if memProfileRate > 0 {
		runtime.MemProfileRate = memProfileRate
	}

	return func() {
		defer pprof.StopCPUProfile()
		defer trace.Stop()
	}, nil
}

func saveMemoryProfile(prefix string) {
	mem, err := os.Create(fmt.Sprintf("%s.%v.mem", prefix, os.Getpid()))
	if err != nil {
		panic(err)
	}
	if err = pprof.WriteHeapProfile(mem); err != nil {
		panic(err)
	}
}

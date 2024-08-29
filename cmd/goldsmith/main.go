package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/brandonpollack23/goldsmith/pkg/fft"
	"github.com/brandonpollack23/goldsmith/pkg/vis"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/wav"
	"github.com/spf13/cobra"
)

// TODO get some of buffer and perform FFT in buckets, aim for 60 fps and pick the time to make each FFT based on that.
// TODO In a goroutine watching the current display time (and waiting or doing a callback based on it) and the FFT channel display buckets using bars with bubbles
// TODO display playback bar at the bottom with timestamp and max time etc.
// TODO Volume with beep
// TODO other beep effects?
// TODO other visualizers and flag to choose
// TODO add CTRL for play/pause

var targetFPS uint32

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

	rootCmd.PersistentFlags().Uint32VarP(&targetFPS, "target_fps", "f", 60,
		"The updates FPS for the visualizer, affects FFT window")

	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v", err)
		os.Exit(1)
	}
}

func runVisualizer(cmd *cobra.Command, args []string) error {
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

	// Initialize the speaker to use the sample rate of the audio file selected.
	// I can also use beep.Resample around the streamer to always use a specific
	// output sample rate for everything no matter the input.
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		return fmt.Errorf("cannot initializer speaker: %w", err)
	}

	windowDuration := time.Duration(float64(time.Second) / float64(targetFPS))
	fftWindowSize := format.SampleRate.N(windowDuration)
	fftStreamer := fft.NewFFTStreamer(streamer, fftWindowSize, format)

	speaker.Play(&fftStreamer)
	visualizer := vis.NewHorizontalBarsVisualizer(64)

	err = loop(&fftStreamer, windowDuration, visualizer)
	return err
}

func loop(s fft.FFTStreamer, windowDuration time.Duration, visualizer vis.Visualizer) error {
	for {
		select {
		case <-time.After(windowDuration):
			fftWindow := <-s.FFTChan()
			visualizer.UpdateVisualizer(vis.NewFFTData{fftWindow.Data})
		}
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

func printTimestamp(streamer beep.StreamSeeker, format beep.Format) {
	speaker.Lock()
	defer speaker.Unlock()

	fmt.Printf("%d.2\n", format.SampleRate.D(streamer.Position()).Round(time.Millisecond))
}

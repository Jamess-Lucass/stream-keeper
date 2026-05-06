package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// cli parameters
var (
	image      string
	key        string
	csvPath    string
	resolution string
)

// bitsPerPixelKbps is the empirically-derived ratio of kbps needed per pixel
// for a still-image H.264 stream at 10fps with tune=stillimage. This value
// sits safely between YouTube's minimum requirements and the recommended
// bitrates for full-motion video, optimized for keeping a stream alive with
// minimal bandwidth.
const bitsPerPixelKbps = 0.00075

// minBitrateKbps is the minimum bitrate floor to ensure stream stability
// at very low resolutions.
const minBitrateKbps = 500

func getBitrateForResolution(width, height int) int {
	pixels := width * height
	bitrate := int(float64(pixels) * bitsPerPixelKbps)
	if bitrate < minBitrateKbps {
		return minBitrateKbps
	}
	return bitrate
}

func parseResolution(res string) (int, int, error) {
	parts := strings.SplitN(res, "x", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid resolution format %q, expected WxH (e.g. 1920x1080)", res)
	}
	w, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid width %q: %w", parts[0], err)
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid height %q: %w", parts[1], err)
	}
	return w, h, nil
}

var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "Stream content to YouTube live streams",
	Long: `Stream an image to one or multiple YouTube live streams using ffmpeg.
You can specify a single stream key or provide a CSV file with multiple keys.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if key == "" && csvPath == "" {
			return fmt.Errorf("must specify either key or csv")
		}

		if key != "" && csvPath != "" {
			return fmt.Errorf("cannot specify both key and csv")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		width, height, err := parseResolution(resolution)
		if err != nil {
			return err
		}
		bitrate := getBitrateForResolution(width, height)
		bitrateStr := fmt.Sprintf("%dk", bitrate)
		bufsizeStr := fmt.Sprintf("%dk", bitrate*2)
		vf := fmt.Sprintf("scale=%d:%d,format=yuv420p", width, height)

		log.Printf("Resolution: %dx%d, Bitrate: %s, Bufsize: %s", width, height, bitrateStr, bufsizeStr)

		keys := []string{}

		if csvPath != "" {
			csvKeys, err := readKeysFromCSV(csvPath)
			if err != nil {
				return err
			}
			keys = append(keys, csvKeys...)
		}

		if key != "" {
			keys = append(keys, key)
		}

		log.Printf("keys: %v", keys)

		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		// Handle shutdown signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			log.Println("Shutting down ffmpeg...")
			cancel()
		}()

		// Build tee muxer output string
		outputs := make([]string, len(keys))
		for i, k := range keys {
			outputs[i] = fmt.Sprintf("[f=flv:onfail=ignore]rtmp://a.rtmp.youtube.com/live2/%s", k)
		}
		teeOutput := strings.Join(outputs, "|")

		ffmpegArgs := []string{
			"-re",
			"-loop", "1",
			"-f", "image2",
			"-i", image,
			"-f", "lavfi",
			"-i", "anullsrc=r=44100:cl=stereo",
			"-vf", vf,
			"-r", "10",
			"-c:v", "libx264",
			"-preset", "veryfast",
			"-tune", "stillimage",
			"-b:v", bitrateStr,
			"-maxrate", bitrateStr,
			"-bufsize", bufsizeStr,
			"-g", "20",
			"-c:a", "aac",
			"-b:a", "128k",
			"-map", "0:v",
			"-map", "1:a",
			"-f", "tee",
			teeOutput,
		}

		// Restart loop: if ffmpeg exits unexpectedly, restart after a short delay
		for {
			ffmpeg := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
			ffmpeg.Stdout = nil
			ffmpeg.Stderr = os.Stderr

			log.Printf("Starting ffmpeg with %d output(s)", len(keys))

			err = ffmpeg.Run()
			if ctx.Err() != nil {
				return nil // graceful shutdown
			}
			log.Printf("ffmpeg exited unexpectedly: %v, restarting in 5s...", err)
			time.Sleep(5 * time.Second)
		}
	},
}

func readKeysFromCSV(path string) (keys []string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening CSV file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("error closing CSV file: %w", cerr)
		}
	}()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV file: %w", err)
	}

	for _, record := range records {
		keys = append(keys, record[0])
	}
	return keys, nil
}

func init() {
	streamCmd.Flags().StringVarP(&image, "image", "i", "", "Path to the image file to stream (required)")
	streamCmd.Flags().StringVarP(&key, "key", "k", "", "Single YouTube stream key")
	streamCmd.Flags().StringVarP(&csvPath, "csv", "c", "", "Path to CSV file with stream keys")
	streamCmd.Flags().StringVarP(&resolution, "resolution", "r", "1920x1080", "Output resolution (e.g. 1920x1080, 1280x720)")

	if err := streamCmd.MarkFlagRequired("image"); err != nil {
		log.Fatalf("error marking flag as required: %v", err)
	}

	rootCmd.AddCommand(streamCmd)
}

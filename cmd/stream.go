package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// cli parameters
var (
	image   string
	key     string
	csvPath string
)

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
		keys := []string{}

		if csvPath != "" {
			file, err := os.Open(csvPath)
			if err != nil {
				return fmt.Errorf("error opening CSV file: %w", err)
			}
			defer file.Close()

			reader := csv.NewReader(file)
			records, err := reader.ReadAll()
			if err != nil {
				return fmt.Errorf("error reading CSV file: %w", err)
			}

			for _, record := range records {
				keys = append(keys, record[0])
			}
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
			log.Println("Shutting down ffmpeg processes...")
			cancel()
		}()

		eg, ctx := errgroup.WithContext(ctx)

		for _, key := range keys {
			eg.Go(func() error {
				ffmpegArgs := []string{
					"-re",
					"-loop", "1",
					"-f", "image2",
					"-i", image,
					"-f", "lavfi",
					"-i", "anullsrc=r=44100:cl=stereo",
					"-vf", "format=yuv420p",
					"-r", "10",
					"-c:v", "libx264",
					"-preset", "veryfast",
					"-tune", "stillimage",
					"-b:v", "1500k",
					"-maxrate", "1500k",
					"-bufsize", "3000k",
					"-g", "300",
					"-c:a", "aac",
					"-b:a", "128k",
					"-reconnect", "1",
					"-reconnect_streamed", "1",
					"-reconnect_delay_max", "5",
					"-f", "flv",
					fmt.Sprintf("rtmp://a.rtmp.youtube.com/live2/%s", key),
				}

				ffmpeg := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
				ffmpeg.Stdout = nil
				ffmpeg.Stderr = nil

				log.Printf("Starting ffmpeg for key %s", key)

				err := ffmpeg.Run()
				if err != nil && ctx.Err() != context.Canceled {
					return fmt.Errorf("failed to run ffmpeg: %w", err)
				}

				return nil
			})
		}

		err := eg.Wait()
		if err != nil {
			log.Printf("Completed with error(s): %v\n", err)
		}

		return nil
	},
}

func init() {
	streamCmd.Flags().StringVarP(&image, "image", "i", "", "Path to the image file to stream (required)")
	streamCmd.Flags().StringVarP(&key, "key", "k", "", "Single YouTube stream key")
	streamCmd.Flags().StringVarP(&csvPath, "csv", "c", "", "Path to CSV file with stream keys")

	streamCmd.MarkFlagRequired("image")

	rootCmd.AddCommand(streamCmd)
}

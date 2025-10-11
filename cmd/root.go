package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xi-mad/MontageGo/internal/ffprobe"
	"github.com/xi-mad/MontageGo/internal/processor"
	"github.com/xi-mad/MontageGo/pkg/config"

	"github.com/spf13/cobra"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "MontageGo [video_file]",
	Short: "MontageGo creates a thumbnail sheet for a video file.",
	Long:  `MontageGo is a smart wrapper for FFmpeg to generate beautiful and informative thumbnail sheets for video files.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Quiet && cfg.Verbose {
			return fmt.Errorf("flags --quiet and --verbose cannot be used together")
		}

		// --quiet is a shorthand for hiding both log types
		if cfg.Quiet {
			cfg.ShowAppLog = false
			cfg.ShowFfmpegLog = false
		}

		cfg.InputPath = args[0]

		return runMontage(cfg)
	},
}

// SetVersion sets the version for the root command.
func SetVersion(version string) {
	rootCmd.Version = version
}

func runMontage(cfg *config.Config) error {
	// Determine the output stream for application logs.
	logWriter := os.Stdout
	if cfg.OutputPath == "-" {
		logWriter = os.Stderr
	}

	if cfg.OutputPath == "" {
		inputDir := filepath.Dir(cfg.InputPath)
		baseName := strings.TrimSuffix(filepath.Base(cfg.InputPath), filepath.Ext(cfg.InputPath))
		newFileName := baseName + "_montage.jpg"
		cfg.OutputPath = filepath.Join(inputDir, newFileName)
	}

	if cfg.ShowAppLog {
		fmt.Fprintln(logWriter, "Analyzing video file:", cfg.InputPath)
	}
	videoInfo, err := ffprobe.GetVideoInfo(cfg.InputPath, cfg.FfprobePath)
	if err != nil {
		return fmt.Errorf("failed to get video info: %w", err)
	}

	if cfg.ShowAppLog {
		fmt.Fprintln(logWriter, "Video analysis complete. Starting montage generation...")
	}
	proc := processor.New(cfg, videoInfo)
	if err := proc.Run(); err != nil {
		return fmt.Errorf("failed to generate montage: %w", err)
	}

	if cfg.ShowAppLog {
		if cfg.OutputPath != "-" {
			fmt.Fprintln(logWriter, "✅ Montage generated successfully at:", cfg.OutputPath)
		} else {
			fmt.Fprintln(logWriter, "✅ Montage generated successfully to stdout.")
		}
	}
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Cobra prints the error for us, so we just need to exit.
		os.Exit(1)
	}
}

func init() {
	// This will be called by main.go to set the version.
	rootCmd.SetVersionTemplate(`{{printf "%s\n" .Version}}`)
	cfg = config.NewConfig()

	// File and Path Flags
	rootCmd.PersistentFlags().StringVarP(&cfg.OutputPath, "output", "o", "", "Output path. Use '-' to stream image data to stdout.")

	rootCmd.PersistentFlags().IntVarP(&cfg.Columns, "columns", "c", 4, "Number of columns in the grid")
	rootCmd.PersistentFlags().IntVarP(&cfg.Rows, "rows", "r", 5, "Number of rows in the grid")
	rootCmd.PersistentFlags().IntVar(&cfg.ThumbWidth, "thumb-width", 640, "Width of each thumbnail")
	rootCmd.PersistentFlags().IntVar(&cfg.ThumbHeight, "thumb-height", -1, "Height of each thumbnail. Defaults to -1 (auto-scale based on width and aspect ratio)")
	rootCmd.PersistentFlags().IntVar(&cfg.Padding, "padding", 5, "Padding between thumbnails")
	rootCmd.PersistentFlags().IntVar(&cfg.Margin, "margin", 20, "Margin around the grid")
	rootCmd.PersistentFlags().IntVar(&cfg.HeaderHeight, "header", 150, "Height of the header section")

	rootCmd.PersistentFlags().StringVar(&cfg.FontFile, "font-file", "", "Path to a .ttf font file for text rendering. If not provided, text will not be rendered.")
	rootCmd.PersistentFlags().StringVar(&cfg.FontColor, "font-color", "white", "Color of the main font")
	rootCmd.PersistentFlags().StringVar(&cfg.ShadowColor, "shadow-color", "black", "Color of the text shadow")
	rootCmd.PersistentFlags().StringVar(&cfg.BackgroundColor, "bg-color", "#222222", "Background color of the montage")

	// New flags for quality and aesthetics
	rootCmd.PersistentFlags().IntVar(&cfg.JpegQuality, "jpeg-quality", 2, "JPEG quality for the output image (1-31, lower is better)")
	rootCmd.PersistentFlags().IntVar(&cfg.BorderThickness, "border-thickness", 1, "Thickness of the border around each thumbnail")
	rootCmd.PersistentFlags().StringVar(&cfg.BorderColor, "border-color", "#111111", "Color of the border around each thumbnail")

	// Paths for external binaries
	rootCmd.PersistentFlags().StringVar(&cfg.FfmpegPath, "ffmpeg-path", "ffmpeg", "Path to the ffmpeg executable")
	rootCmd.PersistentFlags().StringVar(&cfg.FfprobePath, "ffprobe-path", "ffprobe", "Path to the ffprobe executable")

	// Log level flags
	rootCmd.PersistentFlags().BoolVarP(&cfg.Quiet, "quiet", "q", false, "Shorthand for --show-app-log=false and --show-ffmpeg-log=false")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Enable verbose output, including the full ffmpeg command")
	rootCmd.PersistentFlags().BoolVar(&cfg.ShowAppLog, "show-app-log", true, "Show application's own log messages (e.g., 'Analyzing...')")
	rootCmd.PersistentFlags().BoolVar(&cfg.ShowFfmpegLog, "show-ffmpeg-log", true, "Show real-time output from the ffmpeg process")
}

package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all the configuration for the MontageGo tool.
type Config struct {
	InputPath       string `yaml:"input_path"`
	OutputPath      string `yaml:"output_path"`
	Columns         int    `yaml:"columns"`
	Rows            int    `yaml:"rows"`
	ThumbWidth      int    `yaml:"thumb_width"`
	ThumbHeight     int    `yaml:"thumb_height"`
	Padding         int    `yaml:"padding"`
	Margin          int    `yaml:"margin"`
	HeaderHeight    int    `yaml:"header_height"`
	FontFile        string `yaml:"font_file"`
	FontColor       string `yaml:"font_color"`
	ShadowColor     string `yaml:"shadow_color"`
	BackgroundColor string `yaml:"background_color"`
	JpegQuality     int    `yaml:"jpeg_quality"`
	FfmpegPath      string `yaml:"ffmpeg_path"`
	FfprobePath     string `yaml:"ffprobe_path"`
	Quiet           bool   `yaml:"quiet"`
	Verbose         bool   `yaml:"verbose"`
	ShowAppLog      bool   `yaml:"show_app_log"`
	ShowFfmpegLog   bool   `yaml:"show_ffmpeg_log"`
}

func NewConfig() *Config {
	return &Config{}
}

// Load reads a YAML config file from the given path and returns a Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

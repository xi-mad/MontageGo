package config

// Config holds all the configuration for the MontageGo tool.
type Config struct {
	InputPath       string
	OutputPath      string
	Columns         int
	Rows            int
	ThumbWidth      int
	ThumbHeight     int
	Padding         int
	Margin          int
	HeaderHeight    int
	FontFile        string
	FontColor       string
	ShadowColor     string
	BackgroundColor string
	JpegQuality     int
	BorderThickness int
	BorderColor     string
	FfmpegPath      string
	FfprobePath     string
	Quiet           bool
	Verbose         bool
	ShowAppLog      bool
	ShowFfmpegLog   bool
}

func NewConfig() *Config {
	return &Config{}
}

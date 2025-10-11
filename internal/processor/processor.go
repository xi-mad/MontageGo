package processor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"strconv"

	"github.com/xi-mad/MontageGo/internal/ffprobe"
	"github.com/xi-mad/MontageGo/pkg/config"
)

type Processor struct {
	Config    *config.Config
	VideoInfo *ffprobe.VideoInfo
}

func New(cfg *config.Config, info *ffprobe.VideoInfo) *Processor {
	return &Processor{
		Config:    cfg,
		VideoInfo: info,
	}
}

// Run orchestrates the montage creation process.
func (p *Processor) Run() error {
	filterComplex, err := p.buildFilterComplex()
	if err != nil {
		return fmt.Errorf("failed to build filter_complex: %w", err)
	}

	args := []string{
		"-hide_banner",
		"-y",
		"-i", p.VideoInfo.Path,
		"-filter_complex", filterComplex,
		"-map", "[v_out]",
		"-q:v", fmt.Sprintf("%d", p.Config.JpegQuality),
	}

	// Add output-specific arguments. If output path is "-", stream to stdout.
	if p.Config.OutputPath == "-" {
		// The "mjpeg" format is suitable for streaming JPEG images to a pipe.
		args = append(args, "-f", "mjpeg", "pipe:1")
	} else {
		// Otherwise, write to the specified file path.
		args = append(args, p.Config.OutputPath)
	}

	cmd := exec.Command(p.Config.FfmpegPath, args...)

	// Determine the output stream for application logs.
	logWriter := os.Stdout
	if p.Config.OutputPath == "-" {
		logWriter = os.Stderr
	}

	if p.Config.Verbose {
		fullCmd := p.Config.FfmpegPath + " " + strings.Join(args, " ")
		fmt.Fprintln(logWriter, "\nExecuting FFmpeg command:")
		fmt.Fprintln(logWriter, fullCmd)
		fmt.Fprintln(logWriter)
	}

	var stderr bytes.Buffer
	// If ffmpeg log is hidden, capture stderr for potential error reporting.
	// Otherwise, stream it directly to the user's terminal (os.Stderr).
	if !p.Config.ShowFfmpegLog {
		cmd.Stderr = &stderr
	} else {
		// IMPORTANT: ffmpeg's stdout may be image data.
		// Its progress/log output (which goes to its stderr) must go to our stderr.
		cmd.Stderr = os.Stderr
	}

	// If output path is a file, ffmpeg writes to it directly.
	// If output path is "-", we need to pipe ffmpeg's stdout to our stdout.
	if p.Config.OutputPath == "-" {
		cmd.Stdout = os.Stdout
	}

	err = cmd.Run()
	if err != nil {
		// If ffmpeg log was hidden and an error occurred, print the captured stderr.
		if !p.Config.ShowFfmpegLog && stderr.Len() > 0 {
			return fmt.Errorf("ffmpeg error: %v\n--- FFMPEG OUTPUT ---\n%s", err, stderr.String())
		}
		return err
	}
	return nil
}

func (p *Processor) buildFilterComplex() (string, error) {
	var filters []string
	currentStream := "[0:v]"

	// 1. Trim video to 90% of duration to avoid intros/outros.
	trimmedStream, trimmedDuration, trimFilter := p.buildTrimFilter(currentStream)
	filters = append(filters, trimFilter)
	currentStream = trimmedStream

	// 2. Select frames and scale them.
	framesStream, framesFilter := p.buildFramesFilter(currentStream, trimmedDuration)
	filters = append(filters, framesFilter)
	currentStream = framesStream

	// 3. Add a border to each thumbnail if specified.
	borderedStream, borderFilter := p.buildBorderFilter(currentStream)
	if borderFilter != "" {
		filters = append(filters, borderFilter)
		currentStream = borderedStream
	}

	// 4. Tile the frames into a grid.
	gridStream, tileFilter := p.buildTileFilter(currentStream)
	filters = append(filters, tileFilter)
	currentStream = gridStream

	// 5. Pad the grid to add background, header, and margins.
	paddedStream, padFilter, isPadded := p.buildPadFilter(currentStream)
	filters = append(filters, padFilter)
	if isPadded {
		currentStream = paddedStream
	}

	// 6. Draw text if a font file is provided.
	textFilters := p.buildTextFilters(currentStream)
	filters = append(filters, textFilters...)

	return strings.Join(filters, ";"), nil
}

// buildTrimFilter creates the filter to trim the video to 90% of its duration.
func (p *Processor) buildTrimFilter(input string) (output string, duration float64, filter string) {
	duration = p.VideoInfo.Duration * 0.9
	start := p.VideoInfo.Duration * 0.05
	output = "[trimmed]"
	filter = fmt.Sprintf("%strim=start=%.4f:duration=%.4f,setpts=PTS-STARTPTS%s", input, start, duration, output)
	return
}

// buildFramesFilter creates the filter to select and scale frames.
func (p *Processor) buildFramesFilter(input string, duration float64) (output, filter string) {
	numFrames := p.Config.Columns * p.Config.Rows
	fps := float64(numFrames) / duration
	output = "[frames]"
	filter = fmt.Sprintf("%sfps=%.6f,scale=%d:%d%s", input, fps, p.Config.ThumbWidth, p.Config.ThumbHeight, output)
	return
}

// buildBorderFilter creates the filter to add a border to each frame.
func (p *Processor) buildBorderFilter(input string) (output, filter string) {
	if p.Config.BorderThickness <= 0 {
		return input, "" // Return original input stream and no filter
	}
	output = "[bordered_frames]"
	filter = fmt.Sprintf(
		"%sdrawbox=x=0:y=0:w=%d:h=%d:color=%s:t=%d%s",
		input, p.Config.ThumbWidth, p.Config.ThumbHeight, p.Config.BorderColor, p.Config.BorderThickness, output,
	)
	return
}

// buildTileFilter creates the filter to arrange frames in a grid.
func (p *Processor) buildTileFilter(input string) (output, filter string) {
	output = "[grid]"
	filter = fmt.Sprintf("%stile=%dx%d:padding=%d:margin=0%s", input, p.Config.Columns, p.Config.Rows, p.Config.Padding, output)
	return
}

// buildPadFilter creates the filter to add padding, header, and background.
func (p *Processor) buildPadFilter(input string) (output, filter string, wasPadded bool) {
	output = "[padded_grid]"
	if p.Config.FontFile == "" {
		output = "[v_out]" // This is the final stream if no text is drawn
	}

	filter = fmt.Sprintf(
		"%spad=width=iw+%d:height=ih+%d:x=%d:y=%d:color=%s%s",
		input,
		2*p.Config.Margin,
		p.Config.HeaderHeight+2*p.Config.Margin,
		p.Config.Margin,
		p.Config.HeaderHeight+p.Config.Margin,
		p.Config.BackgroundColor,
		output,
	)
	wasPadded = true
	return
}

// buildTextFilters creates the filters for drawing all text elements.
func (p *Processor) buildTextFilters(input string) []string {
	if p.Config.FontFile == "" {
		// If there's no font file, the input must be mapped to the output.
		// However, buildPadFilter already handled this by naming its output [v_out].
		return nil
	}

	var textFilters []string
	currentStream := input
	escapedFontFile := strings.ReplaceAll(p.Config.FontFile, "\\", "/")
	escapedFontFile = strings.ReplaceAll(escapedFontFile, ":", "\\\\:")

	// Draw filename with dynamic font size to prevent overflow
	filename := filepath.Base(p.VideoInfo.Path)
	filename = p.escapeFFmpegDrawtext(filename)

	// The fontsize expression dynamically scales the font to fit the width.
	// `min(40, ...)` sets a max font size of 40.
	// `(w*0.9/text_w)*40` scales the font down if the text (at size 40) is wider than 90% of the canvas.
	filenameFontSize := "min(40, (w*0.9/text_w)*40)"

	filenameFilter := p.drawTextFilterWithCustomFontsize(
		filename, escapedFontFile, p.Config.FontColor, filenameFontSize,
		"(w-tw)/2", "30",
		currentStream, "[with_filename]",
	)
	textFilters = append(textFilters, filenameFilter)
	currentStream = "[with_filename]"

	// Draw metadata line 1
	metadata1 := p.formatMetadataLine1()
	metadata1 = p.escapeFFmpegDrawtext(metadata1)
	meta1Filter := p.drawTextFilter(
		metadata1, escapedFontFile, "#cccccc", 20,
		"(w-tw)/2", "80",
		currentStream, "[with_meta1]",
	)
	textFilters = append(textFilters, meta1Filter)
	currentStream = "[with_meta1]"

	// Draw metadata line 2
	metadata2 := p.formatMetadataLine2()
	metadata2 = p.escapeFFmpegDrawtext(metadata2)
	meta2Filter := p.drawTextFilter(
		metadata2, escapedFontFile, "#cccccc", 20,
		"(w-tw)/2", "105",
		currentStream, "[v_out]", // Final output stream
	)
	textFilters = append(textFilters, meta2Filter)

	return textFilters
}

// formatMetadataLine1 generates the first line of metadata: Resolution | FPS | Bitrate
func (p *Processor) formatMetadataLine1() string {
	// Dimensions
	dims := fmt.Sprintf("%dx%d", p.VideoInfo.Width, p.VideoInfo.Height)

	// Frame rate
	var fpsStr string
	if p.VideoInfo.AvgFrameRate != "" {
		parts := strings.Split(p.VideoInfo.AvgFrameRate, "/")
		if len(parts) == 2 {
			if num, err := strconv.ParseFloat(parts[0], 64); err == nil {
				if den, err := strconv.ParseFloat(parts[1], 64); err == nil && den != 0 {
					fpsStr = fmt.Sprintf("%.2f FPS", num/den)
				}
			}
		}
	}
	if fpsStr == "" {
		fpsStr = "N/A FPS"
	}

	// Bitrate
	var bitrateStr string
	if bitRate, err := strconv.ParseFloat(p.VideoInfo.BitRate, 64); err == nil {
		bitrateMbps := bitRate / 1000000
		bitrateStr = fmt.Sprintf("%.2f Mbps", bitrateMbps)
	} else {
		bitrateStr = "N/A Mbps"
	}

	return fmt.Sprintf("%s | %s | %s", dims, fpsStr, bitrateStr)
}

// formatMetadataLine2 generates the second line of metadata: Duration | File Size | Codecs
func (p *Processor) formatMetadataLine2() string {
	// Duration
	d := time.Duration(p.VideoInfo.Duration * float64(time.Second))
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	durationStr := fmt.Sprintf("%02d:%02d:%02d", h, m, s)

	// File size
	sizeMB := float64(p.VideoInfo.FileSize) / (1024 * 1024)
	sizeStr := fmt.Sprintf("%.2f MB", sizeMB)

	// Codecs
	codecs := strings.ToUpper(p.VideoInfo.VideoCodec)
	if p.VideoInfo.AudioCodec != "" {
		codecs += " / " + strings.ToUpper(p.VideoInfo.AudioCodec)
	}

	return fmt.Sprintf("%s | %s | %s", durationStr, sizeStr, codecs)
}

// escapeFFmpegDrawtext escapes characters that are special to ffmpeg's drawtext filter.
func (p *Processor) escapeFFmpegDrawtext(text string) string {
	// The characters ' \ : % need to be escaped with a backslash.
	// We use a replacer for efficiency and correctness.
	r := strings.NewReplacer(
		"\\", "\\\\", // Backslash must be escaped first
		"'", `\'`,
		":", `\:`,
		"%", `\%`,
	)
	return r.Replace(text)
}

func (p *Processor) drawTextFilter(text, fontfile, color string, size int, x, y, input, output string) string {
	fontSizeStr := fmt.Sprintf("%d", size)
	return p.drawTextFilterWithCustomFontsize(text, fontfile, color, fontSizeStr, x, y, input, output)
}

func (p *Processor) drawTextFilterWithCustomFontsize(text, fontfile, color, size, x, y, input, output string) string {
	return fmt.Sprintf(
		"%sdrawtext=fontfile='%s':text='%s':fontcolor=%s:fontsize='%s':x=%s:y=%s:shadowcolor=%s:shadowx=2:shadowy=2%s",
		input, fontfile, text, color, size, x, y, p.Config.ShadowColor, output,
	)
}

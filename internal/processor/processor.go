package processor

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fogleman/gg"
	"github.com/xi-mad/MontageGo/internal/ffprobe"
	"github.com/xi-mad/MontageGo/pkg/config"
)

var (
	colorNameToHex = map[string]string{
		"black":     "#000000",
		"white":     "#FFFFFF",
		"red":       "#FF0000",
		"lime":      "#00FF00",
		"blue":      "#0000FF",
		"yellow":    "#FFFF00",
		"cyan":      "#00FFFF",
		"magenta":   "#FF00FF",
		"silver":    "#C0C0C0",
		"gray":      "#808080",
		"grey":      "#808080",
		"maroon":    "#800000",
		"olive":     "#808000",
		"green":     "#008000",
		"purple":    "#800080",
		"teal":      "#008080",
		"navy":      "#000080",
		"darkgray":  "#A9A9A9",
		"darkgrey":  "#A9A9A9",
		"lightgray": "#D3D3D3",
		"lightgrey": "#D3D3D3",
	}
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
	// Pre-calculate thumbnail dimensions, especially for auto-height.
	thumbWidth := p.Config.ThumbWidth
	thumbHeight := p.Config.ThumbHeight
	if thumbHeight <= 0 {
		// Ensure we don't divide by zero if video info is weird.
		if p.VideoInfo.Height == 0 {
			return fmt.Errorf("video height is 0, cannot auto-calculate thumbnail height")
		}
		thumbHeight = int(float64(thumbWidth) / (float64(p.VideoInfo.Width) / float64(p.VideoInfo.Height)))
	}

	// 1. Calculate timestamps and extract frames in parallel into memory.
	frames, timestamps, err := p.extractFrames(thumbWidth, thumbHeight)
	if err != nil {
		return fmt.Errorf("failed to extract frames: %w", err)
	}

	// 2. Compose the final image using gg.
	err = p.composeMontage(frames, timestamps, thumbWidth, thumbHeight)
	if err != nil {
		return fmt.Errorf("failed to compose montage: %w", err)
	}

	return nil
}

// extractFrames calculates timestamps and extracts video frames into memory.
// This new version uses a single ffmpeg process to extract all frames at once
// for much better efficiency than spawning a process per frame.
func (p *Processor) extractFrames(thumbWidth, thumbHeight int) ([]image.Image, []float64, error) {
	numFrames := p.Config.Columns * p.Config.Rows
	if numFrames <= 0 {
		return nil, nil, fmt.Errorf("number of frames must be positive")
	}

	// Use 90% of the video duration, skipping the first and last 5%.
	duration := p.VideoInfo.Duration * 0.9
	startOffset := p.VideoInfo.Duration * 0.05
	interval := duration / float64(numFrames)

	timestamps := make([]float64, numFrames)
	for i := 0; i < numFrames; i++ {
		timestamps[i] = startOffset + (float64(i) * interval)
	}

	// --- Efficient frame extraction using a single ffmpeg process ---

	// 1. Get video FPS.
	var fps float64 = 25.0 // Default if not found
	if p.VideoInfo.AvgFrameRate != "" {
		parts := strings.Split(p.VideoInfo.AvgFrameRate, "/")
		if len(parts) == 2 {
			if num, err := strconv.ParseFloat(parts[0], 64); err == nil {
				if den, err := strconv.ParseFloat(parts[1], 64); err == nil && den != 0 {
					fps = num / den
				}
			}
		}
	}

	// 2. Generate the 'select' filter string based on frame numbers.
	// We use -ss to seek, so timestamps for frames are relative to startOffset.
	selectParts := make([]string, numFrames)
	for i := 0; i < numFrames; i++ {
		relativeTimestamp := float64(i) * interval
		frameNumber := int(relativeTimestamp * fps)
		// The comma in "eq(n,123)" must be escaped for the ffmpeg filter parser.
		selectParts[i] = fmt.Sprintf("eq(n\\,%d)", frameNumber)
	}
	selectFilter := "select='" + strings.Join(selectParts, "+") + "'"

	// 3. Construct the ffmpeg command.
	// -ss is before -i for fast seeking.
	// The output is a raw pipe of concatenated JPEG images.
	args := []string{
		"-ss", fmt.Sprintf("%.4f", startOffset),
		"-i", p.VideoInfo.Path,
		"-vf", fmt.Sprintf("%s,scale=%d:%d", selectFilter, thumbWidth, thumbHeight),
		"-vframes", strconv.Itoa(numFrames),
		"-q:v", fmt.Sprintf("%d", p.Config.JpegQuality),
		"-f", "image2pipe",
		"-c:v", "mjpeg",
		"pipe:1",
	}

	cmd := exec.Command(p.Config.FfmpegPath, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, nil, fmt.Errorf("failed to execute ffmpeg: %w\nStderr: %s", err, stderr.String())
	}

	// 4. Decode the concatenated JPEG stream from stdout.
	imageData := out.Bytes()
	jpegSOI := []byte{0xff, 0xd8} // Start of Image
	jpegEOI := []byte{0xff, 0xd9} // End of Image

	frames := make([]image.Image, numFrames)
	var wg sync.WaitGroup
	errs := make(chan error, numFrames)

	searchPos := 0
	frameIndex := 0

	for frameIndex < numFrames {
		// Find the start of the next JPEG image.
		soi := bytes.Index(imageData[searchPos:], jpegSOI)
		if soi == -1 {
			break // No more images found.
		}
		soi += searchPos

		// Find the end of this JPEG image.
		eoi := bytes.Index(imageData[soi:], jpegEOI)
		if eoi == -1 {
			break // Incomplete image data.
		}
		eoi += soi

		imgData := imageData[soi : eoi+2]
		searchPos = eoi + 2

		wg.Add(1)
		go func(index int, data []byte) {
			defer wg.Done()
			img, _, err := image.Decode(bytes.NewReader(data))
			if err != nil {
				errs <- fmt.Errorf("failed to decode frame %d: %w", index, err)
				return
			}
			frames[index] = img
		}(frameIndex, imgData)

		frameIndex++
	}

	wg.Wait()
	close(errs)

	// Check for any errors during decoding.
	for err := range errs {
		return nil, nil, err // Return on the first error.
	}

	// If we didn't find enough frames, it's an error.
	if frameIndex != numFrames {
		return nil, nil, fmt.Errorf("ffmpeg produced %d frames, but %d were expected. Stderr:\n%s", frameIndex, numFrames, stderr.String())
	}

	return frames, timestamps, nil
}

// composeMontage creates the final image by arranging the extracted frames.
func (p *Processor) composeMontage(frames []image.Image, timestamps []float64, thumbWidth, thumbHeight int) error {
	// Dimensions are now passed in.
	gridWidth := p.Config.Columns*thumbWidth + (p.Config.Columns-1)*p.Config.Padding
	gridHeight := p.Config.Rows*thumbHeight + (p.Config.Rows-1)*p.Config.Padding

	totalWidth := gridWidth + 2*p.Config.Margin
	totalHeight := gridHeight + 2*p.Config.Margin + p.Config.HeaderHeight

	dc := gg.NewContext(totalWidth, totalHeight)

	// Draw background
	bgColor, err := parseHexColor(p.Config.BackgroundColor)
	if err != nil {
		return fmt.Errorf("invalid background color: %w", err)
	}
	dc.SetColor(bgColor)
	dc.Clear()

	// Draw header text
	if p.Config.FontFile != "" {
		if err := p.drawText(dc, totalWidth); err != nil {
			return fmt.Errorf("failed to draw text: %w", err)
		}
	}

	// Prepare for drawing timestamps on frames
	var timestampFontColor color.Color
	var timestampShadowColor color.Color
	if p.Config.FontFile != "" {
		if err := dc.LoadFontFace(p.Config.FontFile, 18); err != nil {
			return fmt.Errorf("could not load fontface for timestamp: %w", err)
		}
		timestampFontColor, _ = parseHexColor("white")
		timestampShadowColor, _ = parseHexColor("black")
	}

	// Draw frames
	for i, img := range frames {
		if img == nil {
			continue // Should not happen with current error handling, but good practice.
		}

		row := i / p.Config.Columns
		col := i % p.Config.Columns

		x := p.Config.Margin + col*(thumbWidth+p.Config.Padding)
		y := p.Config.HeaderHeight + p.Config.Margin + row*(thumbHeight+p.Config.Padding)

		dc.DrawImage(img, x, y)

		// Draw timestamp on the frame if font is available
		if p.Config.FontFile != "" {
			timestampStr := formatDuration(timestamps[i])
			// Position is relative to the image's top-left corner
			textX := float64(x + 10)
			textY := float64(y + thumbHeight - 15)

			dc.SetColor(timestampShadowColor)
			dc.DrawStringAnchored(timestampStr, textX+1, textY+1, 0, 1)
			dc.SetColor(timestampFontColor)
			dc.DrawStringAnchored(timestampStr, textX, textY, 0, 1)
		}
	}

	// Save the final image
	// The gg library's JPEG quality is 1-100 (higher is better),
	// while ffmpeg's -q:v is 1-31 (lower is better). We'll do a rough conversion.
	jpegQuality := 100 - (p.Config.JpegQuality-1)*3
	if jpegQuality < 1 {
		jpegQuality = 1
	}
	if jpegQuality > 100 {
		jpegQuality = 100
	}

	if p.Config.OutputPath == "-" {
		return jpeg.Encode(os.Stdout, dc.Image(), &jpeg.Options{Quality: jpegQuality})
	} else {
		return gg.SaveJPG(p.Config.OutputPath, dc.Image(), jpegQuality)
	}
}

// drawText renders the header information onto the montage.
func (p *Processor) drawText(dc *gg.Context, totalWidth int) error {
	// Load font
	if err := dc.LoadFontFace(p.Config.FontFile, 40); err != nil {
		return fmt.Errorf("could not load fontface: %w", err)
	}

	// Shadow color
	shadowColor, err := parseHexColor(p.Config.ShadowColor)
	if err != nil {
		return fmt.Errorf("invalid shadow color: %w", err)
	}

	// Font color
	fontColor, err := parseHexColor(p.Config.FontColor)
	if err != nil {
		return fmt.Errorf("invalid font color: %w", err)
	}

	// --- Draw Filename ---
	filename := filepath.Base(p.VideoInfo.Path)
	// Dynamically adjust font size to fit
	fontSize := 40.0
	for fontSize > 10 {
		if err := dc.LoadFontFace(p.Config.FontFile, fontSize); err != nil {
			return err
		}
		w, _ := dc.MeasureString(filename)
		if w < float64(totalWidth)*0.9 {
			break
		}
		fontSize -= 2
	}
	// Draw shadow then text
	dc.SetColor(shadowColor)
	dc.DrawStringAnchored(filename, float64(totalWidth)/2+2, 30+2, 0.5, 0.5)
	dc.SetColor(fontColor)
	dc.DrawStringAnchored(filename, float64(totalWidth)/2, 30, 0.5, 0.5)

	// --- Draw Metadata Line 1 ---
	if err := dc.LoadFontFace(p.Config.FontFile, 20); err != nil {
		return err
	}
	meta1 := p.formatMetadataLine1()
	dc.SetColor(shadowColor)
	dc.DrawStringAnchored(meta1, float64(totalWidth)/2+1, 80+1, 0.5, 0.5)
	dc.SetColor(color.White) // A lighter color for metadata
	dc.DrawStringAnchored(meta1, float64(totalWidth)/2, 80, 0.5, 0.5)

	// --- Draw Metadata Line 2 ---
	meta2 := p.formatMetadataLine2()
	dc.SetColor(shadowColor)
	dc.DrawStringAnchored(meta2, float64(totalWidth)/2+1, 105+1, 0.5, 0.5)
	dc.SetColor(color.White)
	dc.DrawStringAnchored(meta2, float64(totalWidth)/2, 105, 0.5, 0.5)

	return nil
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
	durationStr := formatDuration(p.VideoInfo.Duration)

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

// formatDuration formats a float64 of seconds into an HH:MM:SS string.
func formatDuration(seconds float64) string {
	d := time.Duration(seconds * float64(time.Second))
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// parseHexColor converts a hex color string (e.g., "#RRGGBB") to a color.Color.
func parseHexColor(s string) (color.Color, error) {
	s = strings.ToLower(s)
	if hex, ok := colorNameToHex[s]; ok {
		s = hex
	}

	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return color.Black, fmt.Errorf("invalid hex color format or unsupported color name: %s", s)
	}
	c, err := strconv.ParseInt(s, 16, 32)
	if err != nil {
		return color.Black, fmt.Errorf("failed to parse hex color: %w", err)
	}
	return color.RGBA{
		R: uint8(c >> 16),
		G: uint8(c >> 8),
		B: uint8(c),
		A: 255,
	}, nil
}

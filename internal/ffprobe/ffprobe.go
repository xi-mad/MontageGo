package ffprobe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// VideoInfo holds simplified, essential video metadata.
type VideoInfo struct {
	Path         string
	Duration     float64
	Width        int
	Height       int
	FileSize     int64
	VideoCodec   string
	AudioCodec   string
	BitRate      string
	AvgFrameRate string
}

// ffprobeOutput matches the JSON structure from the ffprobe command.
type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	CodecType    string `json:"codec_type"`
	CodecName    string `json:"codec_name"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	AvgFrameRate string `json:"avg_frame_rate"`
}

type ffprobeFormat struct {
	Duration string            `json:"duration"`
	Size     string            `json:"size"`
	Filename string            `json:"filename"`
	BitRate  string            `json:"bit_rate"`
	Tags     map[string]string `json:"tags"`
}

// GetVideoInfo executes ffprobe to get video metadata.
func GetVideoInfo(path string, ffprobePath string) (*VideoInfo, error) {
	cmd := exec.Command(ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error running ffprobe: %v: %s", err, stderr.String())
	}

	var ffData ffprobeOutput
	if err := json.Unmarshal(out.Bytes(), &ffData); err != nil {
		return nil, fmt.Errorf("error parsing ffprobe JSON: %w", err)
	}

	info := &VideoInfo{
		Path:    ffData.Format.Filename,
		BitRate: ffData.Format.BitRate,
	}

	if duration, err := strconv.ParseFloat(ffData.Format.Duration, 64); err == nil {
		info.Duration = duration
	}
	if size, err := strconv.ParseInt(ffData.Format.Size, 10, 64); err == nil {
		info.FileSize = size
	}

	for _, stream := range ffData.Streams {
		switch stream.CodecType {
		case "video":
			if info.VideoCodec == "" { // Take the first video stream
				info.Width = stream.Width
				info.Height = stream.Height
				info.VideoCodec = stream.CodecName
				info.AvgFrameRate = stream.AvgFrameRate
			}
		case "audio":
			if info.AudioCodec == "" { // Take the first audio stream
				info.AudioCodec = stream.CodecName
			}
		}
	}

	if info.Width == 0 || info.Height == 0 {
		return nil, fmt.Errorf("could not determine video dimensions from ffprobe output")
	}

	return info, nil
}

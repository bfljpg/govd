package libav

import (
	"bytes"
	"fmt"
	"os"

	"github.com/govdbot/govd/internal/logger"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

// TranscodeToH264 transcodes a video file to H.264/AAC MP4,
// which is universally compatible (iOS, Android, web).
// This is used when the source video uses VP9, AV1, or other
// codecs not supported by iOS Safari / Telegram iOS.
func TranscodeToH264(inputPath string, outputPath string) error {
	logger.L.Debugf("transcoding to H.264: %s -> %s", inputPath, outputPath)

	var stderr bytes.Buffer
	err := ffmpeg.Input(inputPath).
		Output(outputPath, ffmpeg.KwArgs{
			"c:v":      "libx264",
			"c:a":      "aac",
			"crf":      "23",
			"preset":   "fast",
			"movflags": "+faststart",
			"map":      "0",
		}).
		OverWriteOutput().
		WithErrorOutput(&stderr).
		Silent(true).
		Run()

	if err != nil {
		os.Remove(outputPath)
		if stderr.Len() > 0 {
			return fmt.Errorf("failed to transcode to H.264: %w: %s", err, stderr.String())
		}
		return fmt.Errorf("failed to transcode to H.264: %w", err)
	}
	return nil
}

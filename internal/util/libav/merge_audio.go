package libav

import (
	"bytes"
	"fmt"
	"os"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func MergeVideoWithAudio(
	videoPath string,
	audioPath string,
	outputPath string,
) error {
	var stderr bytes.Buffer
	err := ffmpeg.Output(
		[]*ffmpeg.Stream{
			ffmpeg.Input(videoPath),
			ffmpeg.Input(audioPath),
		},
		outputPath,
		ffmpeg.KwArgs{
			"movflags": "+faststart",
			"c:v":      "copy",
			"c:a":      "copy",
		}).
		OverWriteOutput().
		WithErrorOutput(&stderr).
		Silent(true).
		Run()

	if err != nil {
		os.Remove(outputPath)
		if stderr.Len() > 0 {
			return fmt.Errorf("failed to merge files: %w: %s", err, stderr.String())
		}
		return fmt.Errorf("failed to merge files: %w", err)
	}

	return nil
}

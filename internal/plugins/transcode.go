package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util/libav"
)

// TranscodeH264 is a plugin that transcodes VP9/AV1/HEVC videos
// to H.264/AAC for universal iOS compatibility.
var TranscodeH264 = &models.Plugin{
	ID: "transcode_h264",
	RunFunc: func(ctx *models.ExtractorContext, item *models.MediaItem, format *models.DownloadedFormat) error {
		filePath := format.FilePath

		outputPath := strings.TrimSuffix(
			filePath,
			filepath.Ext(filePath),
		) + "_h264.mp4"
		ctx.FilesTracker.Add(outputPath)

		err := libav.TranscodeToH264(filePath, outputPath)
		if err != nil {
			return fmt.Errorf("failed to transcode to H.264: %w", err)
		}

		if err := os.Rename(outputPath, filePath); err != nil {
			return fmt.Errorf("failed to replace original file: %w", err)
		}

		// update codec info so Telegram receives it as a proper video
		format.Format.VideoCodec = database.MediaCodecAvc
		if format.Format.AudioCodec == database.MediaCodecOpus ||
			format.Format.AudioCodec == database.MediaCodecVorbis {
			format.Format.AudioCodec = database.MediaCodecAac
		}

		return nil
	},
}

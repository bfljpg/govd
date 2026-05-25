package youtube

import (
	"fmt"
	"regexp"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
)

var Extractor = &models.Extractor{
	ID:          "youtube",
	DisplayName: "YouTube",

	URLPattern: regexp.MustCompile(`(?:https?:)?(?://)?(?:(?:www|m|music)\.)?(?:youtube(?:-nocookie)?\.com/(?:(?:watch\?(?:.*&)?v=)|(?:embed/)|(?:v/)|(?:shorts/)|(?:live/)|(?:(?:@[^/?#]+)/shorts/))|youtu\.be/)(?P<id>[\w-]{11})(?:[/?&].*)?`),
	Host: []string{
		"youtube",
		"youtu",
		"youtube-nocookie",
	},

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		video, err := GetVideoFromYtDlp(ctx)
		if err != nil {
			return nil, err
		}
		return &models.ExtractorResponse{
			Media: video,
		}, nil
	},
}

func GetVideoFromYtDlp(ctx *models.ExtractorContext) (*models.Media, error) {
	data, err := util.GetYtDlpMetadata(ctx.Context, ctx.ContentURL)
	if err != nil {
		return nil, err
	}

	formats := make([]*models.MediaFormat, 0, len(data.Formats))
	for _, f := range data.Formats {
		if f.URL == "" {
			continue
		}

		vCodec := util.ParseVideoCodec(f.VCodec)
		aCodec := util.ParseAudioCodec(f.ACodec)

		var mediaType database.MediaType
		switch {
		case vCodec != "":
			mediaType = database.MediaTypeVideo
		case aCodec != "":
			mediaType = database.MediaTypeAudio
		default:
			continue
		}

		fileSize := f.Filesize
		if fileSize == 0 {
			fileSize = f.FilesizeApprox
		}

		formats = append(formats, &models.MediaFormat{
			FormatID:   f.FormatID,
			Type:       mediaType,
			VideoCodec: vCodec,
			AudioCodec: aCodec,
			Width:      f.Width,
			Height:     f.Height,
			FileSize:   fileSize,
			Duration:   int32(data.Duration),
			Bitrate:    int64(f.Bitrate),
			URL:        []string{f.URL},
			Title:      data.Title,
			Artist:     data.Uploader,
			DownloadSettings: &models.DownloadSettings{
				ChunkSize: 10 * 1024 * 1024, // 10 MB
			},
		})
	}

	if len(formats) == 0 {
		return nil, fmt.Errorf("no suitable formats found")
	}

	media := ctx.NewMedia()
	media.Caption = data.Description
	item := media.NewItem()
	item.AddFormats(formats...)

	return media, nil
}

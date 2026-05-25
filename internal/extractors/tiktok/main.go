package tiktok

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"github.com/govdbot/govd/internal/util"
)

var VMExtractor = &models.Extractor{
	ID:          "tiktok",
	DisplayName: "TikTok VM",

	URLPattern: regexp.MustCompile(`https:\/\/((?:vm|vt|www)\.)?(vx)?tiktok\.com\/(?:t\/)?(?P<id>[a-zA-Z0-9-]+)`),
	Host:       []string{"tiktok", "vxtiktok"},
	Redirect:   true,

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		redirectURL, err := ctx.FetchLocation(ctx.ContentURL, &networking.RequestParams{
			Headers: webHeaders,
		})
		if err != nil {
			return nil, err
		}
		parsedURL, err := url.Parse(redirectURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redirect url: %w", err)
		}

		if parsedURL.Path == "/404" {
			return nil, util.ErrUnavailable
		}

		if parsedURL.Path == "/login" {
			ctx.Debugf("tiktok is geo restricted in your region, attemping bypass...")
			realURL := parsedURL.Query().Get("redirect_url")
			if realURL == "" {
				return nil, util.ErrGeoRestrictedContent
			}
			return &models.ExtractorResponse{
				URL: realURL,
			}, nil
		}
		return &models.ExtractorResponse{
			URL: redirectURL,
		}, nil
	},
}

var Extractor = &models.Extractor{
	ID:          "tiktok",
	DisplayName: "TikTok",

	URLPattern: regexp.MustCompile(`https?:\/\/((www|m)\.)?(vx)?tiktok\.com\/((?:embed|@[\w\.-]*)\/)?(v(ideo)?|p(hoto)?)\/(?P<id>[0-9]+)`),
	Host:       []string{"tiktok", "vxtiktok"},

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		media, err := GetMedia(ctx)
		if err != nil {
			return nil, err
		}
		return &models.ExtractorResponse{
			URL:   ctx.ContentURL,
			Media: media,
		}, nil
	},
}

func GetMedia(ctx *models.ExtractorContext) (*models.Media, error) {
	var details *WebItemStruct
	var cookies []*http.Cookie
	var err error

	// sometimes web page just returns a
	// login page, so we need to retry
	// a few times to get the correct page
	for range 5 {
		details, cookies, err = GetVideoWeb(ctx)
		if err == nil {
			break
		}
	}
	if err != nil {
		ctx.Warnf("failed to get from web, trying yt-dlp fallback: %v", err)
		ytURL := ctx.ContentURL
		isPhotoPost := strings.Contains(ytURL, "/photo/") || strings.Contains(ytURL, "/p/")
		ytURL = strings.Replace(ytURL, "/photo/", "/video/", 1)
		ytURL = strings.Replace(ytURL, "/p/", "/video/", 1)

		ytDetails, ytErr := util.GetYtDlpMetadata(ctx.Context, ytURL)
		if ytErr != nil {
			return nil, fmt.Errorf("failed to get from web and yt-dlp fallback failed: %w", ytErr)
		}

		media := ctx.NewMedia()
		media.SetCaption(ytDetails.Description)
		if media.Caption == "" {
			media.SetCaption(ytDetails.Title)
		}

		// check if yt-dlp returned image formats (photo slideshow)
		var imageURLs []string
		if isPhotoPost {
			// for photo slideshows, yt-dlp puts images in thumbnails
			seenURLs := make(map[string]bool)
			for _, t := range ytDetails.Thumbnails {
				if t.URL == "" {
					continue
				}
				// yt-dlp sometimes duplicates the same url with different ids (cover, originCover)
				// we only want unique images
				cleanURL := strings.Split(t.URL, "?")[0] // check base url to avoid dupes with different query params
				if !seenURLs[cleanURL] {
					seenURLs[cleanURL] = true
					imageURLs = append(imageURLs, t.URL)
				}
			}
		}

		if len(imageURLs) > 0 {
			// handle as photo slideshow
			for _, imgURL := range imageURLs {
				item := media.NewItem()
				item.AddFormats(&models.MediaFormat{
					Type:     database.MediaTypePhoto,
					FormatID: "image",
					URL:      []string{imgURL},
				})
			}
			return media, nil
		}

		// handle as video
		var bestWidth, bestHeight int32
		var bestBitrate int64
		vcodec := database.MediaCodecAvc
		acodec := database.MediaCodecAac
		for _, f := range ytDetails.Formats {
			parsedVCodec := util.ParseVideoCodec(f.VCodec)
			if parsedVCodec == "" {
				continue
			}
			if int64(f.Bitrate) > bestBitrate || (f.Width > bestWidth && f.Height > bestHeight) {
				bestWidth = f.Width
				bestHeight = f.Height
				bestBitrate = int64(f.Bitrate)
				vcodec = parsedVCodec
				parsedACodec := util.ParseAudioCodec(f.ACodec)
				if parsedACodec != "" {
					acodec = parsedACodec
				}
			}
		}

		item := media.NewItem()
		item.AddFormats(&models.MediaFormat{
			Type:       database.MediaTypeVideo,
			FormatID:   "ytdlp_best",
			URL:        []string{ytURL},
			VideoCodec: vcodec,
			AudioCodec: acodec,
			Width:      bestWidth,
			Height:     bestHeight,
			Duration:   int32(ytDetails.Duration),
			Bitrate:    bestBitrate,
			DownloadSettings: &models.DownloadSettings{
				YtDlpMediaURL: ytURL,
			},
		})
		return media, nil
	}

	media := ctx.NewMedia()
	media.SetCaption(details.Desc)

	isImageSlide := details.ImagePost != nil
	if !isImageSlide {
		item := media.NewItem()
		video := details.Video
		if video.PlayAddr != nil {
			item.AddFormats(&models.MediaFormat{
				Type:       database.MediaTypeVideo,
				FormatID:   video.PlayAddr.URI,
				URL:        video.PlayAddr.URLList,
				VideoCodec: database.MediaCodecAvc,
				AudioCodec: database.MediaCodecAac,
				Width:      video.PlayAddr.Width,
				Height:     video.PlayAddr.Height,
				Duration:   video.Duration,
				DownloadSettings: &models.DownloadSettings{
					// avoid 403 error for videos
					Cookies: cookies,
				},
			})
			return media, nil
		}
		return nil, util.ErrUnavailable
	} else {
		images := details.ImagePost.Images
		for _, image := range images {
			item := media.NewItem()
			item.AddFormats(&models.MediaFormat{
				Type:     database.MediaTypePhoto,
				FormatID: "image",
				URL:      image.URL.URLList,
				DownloadSettings: &models.DownloadSettings{
					Cookies: cookies,
				},
			})
		}
		return media, nil
	}
}


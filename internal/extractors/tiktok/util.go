package tiktok

import (
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/bytedance/sonic"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"github.com/govdbot/govd/internal/util"
)

const (
	videoURLBase = "https://www.tiktok.com/@_/video/"
	photoURLBase = "https://www.tiktok.com/@_/photo/"
)

var (
	universalDataPattern = regexp.MustCompile(`(?s)<script[^>]*\bid=["']__UNIVERSAL_DATA_FOR_REHYDRATION__["'][^>]*>(.*?)<\/script>`)
	photoPathPattern     = regexp.MustCompile(`\/(photo|p)\/[0-9]+`)

	webHeaders = map[string]string{
		"Host":            "www.tiktok.com",
		"Connection":      "keep-alive",
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language": "en-US,en;q=0.9",
		"Sec-Fetch-Dest":  "document",
		"Sec-Fetch-Mode":  "navigate",
		"Sec-Fetch-Site":  "none",
		"Sec-Fetch-User":  "?1",
	}
)

func GetVideoWeb(ctx *models.ExtractorContext) (*WebItemStruct, []*http.Cookie, error) {
	fetchURL := ctx.ContentURL
	if fetchURL == "" {
		awemeID := ctx.ContentID
		urlBase := videoURLBase
		if photoPathPattern.MatchString(ctx.ContentURL) {
			urlBase = photoURLBase
		}
		fetchURL = urlBase + awemeID
	}

	resp, err := ctx.Fetch(
		http.MethodGet,
		fetchURL,
		&networking.RequestParams{
			Headers: webHeaders,
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.Request.URL.Path == "/login" {
		return nil, nil, util.ErrAuthenticationNeeded
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	itemStruct, err := ParseUniversalData(body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse universal data: %w", err)
	}
	return itemStruct, resp.Cookies(), nil
}

func ParseUniversalData(body []byte) (*WebItemStruct, error) {
	matches := universalDataPattern.FindSubmatch(body)
	if len(matches) < 2 {
		return nil, fmt.Errorf("universal data not found")
	}

	var data any
	err := sonic.ConfigFastest.Unmarshal(matches[1], &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal universal data: %w", err)
	}
	logger.WriteFile("tt_universal_data", data)

	defaultScope := util.TraverseJSON(data, "__DEFAULT_SCOPE__")
	if defaultScope == nil {
		return nil, fmt.Errorf("default scope not found")
	}
	logger.WriteFile("tt_default_scope", defaultScope)

	itemStruct := util.TraverseJSON(defaultScope, "itemStruct")
	if itemStruct == nil {
		return nil, util.ErrUnavailable
	}
	logger.WriteFile("tt_item_struct", itemStruct)

	itemStructBytes, err := sonic.ConfigFastest.Marshal(itemStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item struct: %w", err)
	}

	var webItem WebItemStruct
	err = sonic.ConfigFastest.Unmarshal(itemStructBytes, &webItem)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal item struct: %w", err)
	}
	return &webItem, nil
}

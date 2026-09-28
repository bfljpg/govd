package models

import "net/http"

type DownloadSettings struct {
	NumConnections int
	ChunkSize      int64
	Headers        map[string]string
	Cookies        []*http.Cookie
	DecryptionKey  *DecryptionKey
	Retries        int
	YtDlpMediaURL  string // when set, download via yt-dlp instead of HTTP
	YtDlpFormatID  string // when set alongside YtDlpMediaURL, selects a specific yt-dlp format
}

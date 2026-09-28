package util

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/aki237/nscjar"
	"github.com/govdbot/govd/internal/logger"
)

var cookiesCache = make(map[string][]*http.Cookie)

func GetExtractorCookies(extractorID string) []*http.Cookie {
	if extractorID == "" {
		return nil
	}
	cookieFile := extractorID + ".txt"
	return ParseCookieFile(cookieFile)
}

func ParseCookieFile(fileName string) []*http.Cookie {
	cachedCookies, ok := cookiesCache[fileName]
	if ok {
		return cachedCookies
	}

	cookiePath := filepath.Join("private/cookies", fileName)

	cookieFile, err := os.Open(cookiePath)
	if err != nil {
		return nil
	}
	defer cookieFile.Close()

	var parser nscjar.Parser
	cookies, err := parser.Unmarshal(cookieFile)
	if err != nil {
		logger.L.Warnf("failed parsing cookie file %s: %v", fileName, err)
		return nil
	}
	cookiesCache[fileName] = cookies

	logger.L.Debugf("parsed cookie file: %s", fileName)
	return cookies
}

// InvalidateCookieCache removes a cookie file's cached value so the next call
// to ParseCookieFile re-reads it from disk. Used by the keepalive goroutine
// to pick up manually refreshed cookies without restarting the container.
func InvalidateCookieCache(fileName string) error {
	if _, ok := cookiesCache[fileName]; !ok {
		return fmt.Errorf("cookie file %s not in cache", fileName)
	}
	delete(cookiesCache, fileName)
	return nil
}

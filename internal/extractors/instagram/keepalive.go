package instagram

import (
	"fmt"
	"net/http"
	"time"

	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/util"
)

const (
	keepaliveInterval = 6 * time.Hour
	keepaliveEndpoint = "https://i.instagram.com/api/v1/accounts/current_user/?edit=true"
)

// StartKeepalive launches a background goroutine that pings Instagram private
// mobile API every 6 hours to keep the session cookie alive.
func StartKeepalive() {
	go func() {
		logger.L.Info("instagram: session keepalive started (interval: 6h)")
		time.Sleep(30 * time.Second)
		keepalivePing()

		ticker := time.NewTicker(keepaliveInterval)
		defer ticker.Stop()
		for range ticker.C {
			keepalivePing()
		}
	}()
}

func keepalivePing() {
	util.InvalidateCookieCache("instagram.txt")
	cookies := util.ParseCookieFile("instagram.txt")
	if len(cookies) == 0 {
		logger.L.Warn("instagram: keepalive skipped - no cookie file")
		return
	}

	sessionID, csrfToken, dsUID := "", "", ""
	for _, c := range cookies {
		switch c.Name {
		case "sessionid":
			sessionID = c.Value
		case "csrftoken":
			csrfToken = c.Value
		case "ds_user_id":
			dsUID = c.Value
		}
	}

	if sessionID == "" {
		logger.L.Warn("instagram: keepalive skipped - sessionid not found")
		return
	}

	cookieStr := "sessionid=" + sessionID
	if csrfToken != "" {
		cookieStr += "; csrftoken=" + csrfToken
	}
	if dsUID != "" {
		cookieStr += "; ds_user_id=" + dsUID
	}

	req, err := http.NewRequest(http.MethodGet, keepaliveEndpoint, nil)
	if err != nil {
		logger.L.Warnf("instagram: keepalive build failed: %v", err)
		return
	}
	req.Header.Set("User-Agent", "Instagram 275.0.0.27.98 Android (33/13; 420dpi; 1080x2400; samsung; SM-G991B; o1s; exynos2100; en_US; 458229258)")
	req.Header.Set("X-IG-App-ID", "936619743392459")
	req.Header.Set("Cookie", cookieStr)
	if csrfToken != "" {
		req.Header.Set("X-CSRFToken", csrfToken)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.L.Warnf("instagram: keepalive request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		logger.L.Debug("instagram: session keepalive OK")
	} else {
		logger.L.Warnf("instagram: session keepalive returned %s - cookie may need renewal", resp.Status)
	}
}

// SessionStatus returns a human-readable status for the current Instagram session.
func SessionStatus() string {
	cookies := util.ParseCookieFile("instagram.txt")
	for _, c := range cookies {
		if c.Name == "sessionid" && c.Value != "" {
			return fmt.Sprintf("active (sessionid: %s...)", c.Value[:8])
		}
	}
	return "no session"
}

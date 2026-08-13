package networking

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"
)

// ipv4Dialer forces IPv4-only connections to prevent failures on hosts
// without IPv6 connectivity (e.g. where Instagram/Meta CDNs resolve AAAA).
// Using "tcp4" only for non-loopback addresses keeps Docker's internal
// DNS resolver (127.0.0.11) working normally.
var ipv4Dialer = &net.Dialer{
	Timeout:   defaultTimeout,
	KeepAlive: defaultTimeout,
}

func dialIPv4(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, _ := net.SplitHostPort(addr)
	// Allow Docker's internal embedded DNS resolver through unchanged.
	// Everything else forces IPv4 to avoid hanging on unreachable IPv6 addresses.
	if host == "127.0.0.11" {
		return ipv4Dialer.DialContext(ctx, network, addr)
	}
	return ipv4Dialer.DialContext(ctx, "tcp4", addr)
}

func NewTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 proxyFromEnv,
		DialContext:           dialIPv4,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       100,
		ResponseHeaderTimeout: 15 * time.Second,
		DisableCompression:    false,
	}
}

func NewTransportNoProxyFromEnv() *http.Transport {
	transport := NewTransport()
	transport.Proxy = func(_ *http.Request) (*url.URL, error) {
		return nil, nil
	}
	return transport
}

func NewTransportWithProxy(proxyURL *url.URL) *http.Transport {
	transport := NewTransport()
	transport.Proxy = http.ProxyURL(proxyURL)
	return transport
}

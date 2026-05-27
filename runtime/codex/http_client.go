//go:build linux

package codex

import (
	"net"
	"net/http"
	"time"
)

func NewHTTPClient(responseHeaderTimeoutValues ...time.Duration) *http.Client {
	return newCodexHTTPClient(responseHeaderTimeoutValues...)
}

func newCodexHTTPClient(responseHeaderTimeoutValues ...time.Duration) *http.Client {
	transport, _ := http.DefaultTransport.(*http.Transport)
	if transport == nil {
		return &http.Client{}
	}
	responseHeaderTimeout := time.Duration(0)
	if len(responseHeaderTimeoutValues) > 0 {
		responseHeaderTimeout = responseHeaderTimeoutValues[0]
	}
	if responseHeaderTimeout <= 0 {
		responseHeaderTimeout = 90 * time.Second
	}
	clone := transport.Clone()
	clone.ResponseHeaderTimeout = responseHeaderTimeout
	clone.TLSHandshakeTimeout = 10 * time.Second
	clone.ExpectContinueTimeout = time.Second
	clone.DialContext = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext
	return &http.Client{Transport: clone}
}

package todoist

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"orivo/src/systems/logx"
)

// logRequest is swapped out in tests so they do not write to the real log.
var logRequest = logx.Info

var (
	lastMu      sync.Mutex
	lastRequest time.Time
)

// logged writes every request orivo sends to Todoist to the log, with its
// answer, how long it took, and the gap since the previous request.
type logged struct {
	next http.RoundTripper
}

func newHTTP() *http.Client {
	return &http.Client{Timeout: 20 * time.Second, Transport: logged{http.DefaultTransport}}
}

func (l logged) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	lastMu.Lock()
	gap := "first request"
	if !lastRequest.IsZero() {
		gap = start.Sub(lastRequest).Round(time.Millisecond).String() + " since the previous request"
	}
	lastRequest = start
	lastMu.Unlock()

	target := req.URL.Host + req.URL.Path
	if req.URL.RawQuery != "" {
		query, err := url.QueryUnescape(req.URL.RawQuery)
		if err != nil {
			query = req.URL.RawQuery
		}
		target += "?" + query
	}

	resp, err := l.next.RoundTrip(req)
	took := time.Since(start).Round(time.Millisecond)
	if err != nil {
		logRequest("todoist: %s %s failed after %s (%s): %v", req.Method, target, took, gap, err)
		return resp, err
	}
	logRequest("todoist: %s %s answered %s in %s (%s)", req.Method, target, resp.Status, took, gap)
	return resp, nil
}

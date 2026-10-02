package todoist

import (
	"net/http"
	"net/url"
	"time"

	"orivo/src/systems/logx"
)

// logRequest is swapped out in tests so they do not write to the real log.
var logRequest = logx.Info

// logged writes every request orivo sends to Todoist to the log, with its
// answer and how long it took.
type logged struct {
	next http.RoundTripper
}

func newHTTP() *http.Client {
	return &http.Client{Timeout: 20 * time.Second, Transport: logged{http.DefaultTransport}}
}

func (l logged) RoundTrip(req *http.Request) (*http.Response, error) {
	target := req.URL.Host + req.URL.Path
	if req.URL.RawQuery != "" {
		query, err := url.QueryUnescape(req.URL.RawQuery)
		if err != nil {
			query = req.URL.RawQuery
		}
		target += "?" + query
	}

	start := time.Now()
	resp, err := l.next.RoundTrip(req)
	took := time.Since(start).Round(time.Millisecond)
	if err != nil {
		logRequest("todoist: %s %s failed after %s: %v", req.Method, target, took, err)
		return resp, err
	}
	logRequest("todoist: %s %s answered %s in %s", req.Method, target, resp.Status, took)
	return resp, nil
}

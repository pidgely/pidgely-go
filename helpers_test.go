package pidgely

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

const testKey = "pdg_test_0123456789abcdef"

type recorded struct {
	Method string
	Path   string // escaped path
	Query  string
	Header http.Header
	Body   []byte
}

func (r recorded) JSON(t *testing.T) any {
	t.Helper()
	var v any
	if len(r.Body) == 0 {
		return nil
	}
	if err := json.Unmarshal(r.Body, &v); err != nil {
		t.Fatalf("request body is not JSON: %v (%s)", err, r.Body)
	}
	return v
}

type reply struct {
	status  int
	body    string
	headers map[string]string
	hang    bool // block until the request context ends
}

type fakeAPI struct {
	t       *testing.T
	mu      sync.Mutex
	replies []reply
	calls   []recorded
	server  *httptest.Server
	sleeps  []time.Duration
}

// newAPI starts a server answering with the queued replies (then 200 {}).
func newAPI(t *testing.T, replies ...reply) (*fakeAPI, *Client) {
	t.Helper()
	api := &fakeAPI{t: t, replies: replies}
	api.server = httptest.NewServer(http.HandlerFunc(api.handle))
	t.Cleanup(api.server.Close)
	c := NewClient(testKey, WithBaseURL(api.server.URL+"/"))
	c.sleep = func(ctx context.Context, d time.Duration) error {
		api.mu.Lock()
		api.sleeps = append(api.sleeps, d)
		api.mu.Unlock()
		return ctx.Err()
	}
	return api, c
}

func (a *fakeAPI) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	a.mu.Lock()
	a.calls = append(a.calls, recorded{Method: r.Method, Path: r.URL.EscapedPath(), Query: r.URL.RawQuery, Header: r.Header.Clone(), Body: body})
	rep := reply{status: 200, body: "{}"}
	if len(a.replies) > 0 {
		rep = a.replies[0]
		a.replies = a.replies[1:]
	}
	a.mu.Unlock()
	if rep.hang {
		<-r.Context().Done()
		return
	}
	for k, v := range rep.headers {
		w.Header().Set(k, v)
	}
	if rep.status == 0 {
		rep.status = 200
	}
	w.WriteHeader(rep.status)
	_, _ = io.WriteString(w, rep.body)
}

func (a *fakeAPI) last() recorded {
	a.t.Helper()
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.calls) == 0 {
		a.t.Fatal("no request was made")
	}
	return a.calls[len(a.calls)-1]
}

func (a *fakeAPI) all() []recorded {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]recorded(nil), a.calls...)
}

func (a *fakeAPI) slept() []time.Duration {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]time.Duration(nil), a.sleeps...)
}

func (a *fakeAPI) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.calls)
}

func mustEqualJSON(t *testing.T, got any, want string) {
	t.Helper()
	var w any
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("bad want JSON: %v", err)
	}
	g, _ := json.Marshal(got)
	wb, _ := json.Marshal(w)
	if string(g) != string(wb) {
		t.Fatalf("JSON mismatch\n got: %s\nwant: %s", g, wb)
	}
}

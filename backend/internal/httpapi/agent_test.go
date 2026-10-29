package httpapi

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/auth"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/platform/configcenter"
)

func agentTestHandler(t *testing.T, target string) http.Handler {
	t.Helper()
	proxy, err := NewAgentProxy(target)
	if err != nil {
		t.Fatal(err)
	}
	return NewServer(Services{Auth: auth.New(&identityRepo{}), Agent: proxy}, configcenter.NewMemoryCenter(nil), slog.New(slog.NewTextHandler(io.Discard, nil))).Routes()
}

func TestAgentAuthAndUpstreamFailure(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(404)
	}))
	handler := agentTestHandler(t, upstream.URL)
	for _, token := range []string{"", "invalid", "merchant-token"} {
		req := httptest.NewRequest("GET", "/api/v1/agent/events", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Agent-Account-ID", "forged-user")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		want := 401
		if token == "merchant-token" {
			want = 403
		}
		if res.Code != want {
			t.Fatalf("token %q: status=%d", token, res.Code)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("unauthorized request reached Node")
	}
	upstream.Close()
	req := httptest.NewRequest("GET", "/api/v1/agent/events", nil)
	req.Header.Set("Authorization", "Bearer user-token")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != 502 || !strings.Contains(res.Body.String(), `"code":"agent_unavailable"`) {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestAgentStreamsBeforeCompletionAndPropagatesDisconnect(t *testing.T) {
	for _, disconnect := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "disconnect"}[disconnect], func(t *testing.T) {
			headers := make(chan http.Header, 1)
			release := make(chan struct{})
			canceled := make(chan struct{})
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				headers <- r.Header.Clone()
				if r.URL.Path != "/api/v1/agent/events" || r.URL.RawQuery != "probe=1" {
					t.Error("proxy changed request path or query")
				}
				w.Header().Set("Content-Type", "text/event-stream")
				io.WriteString(w, "event: status\ndata: {}\n\n")
				w.(http.Flusher).Flush()
				select {
				case <-release:
					io.WriteString(w, "event: done\ndata: {}\n\n")
				case <-r.Context().Done():
					close(canceled)
				}
			}))
			defer upstream.Close()
			api := httptest.NewServer(agentTestHandler(t, upstream.URL))
			defer api.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			req, _ := http.NewRequestWithContext(ctx, "GET", api.URL+"/api/v1/agent/events?probe=1", nil)
			req.Header.Set("Authorization", "Bearer user-token")
			req.Header.Set("Cookie", "private=value")
			req.Header.Set("X-Agent-Account-ID", "forged-user")
			req.Header.Set("Forwarded", "for=forged")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != 200 || res.Header.Get("Content-Type") != "text/event-stream" {
				t.Fatalf("unexpected response: %d %v", res.StatusCode, res.Header)
			}
			// Upstream cannot finish until this test receives the first event.
			reader := bufio.NewReader(res.Body)
			line, err := reader.ReadString('\n')
			if err != nil || line != "event: status\n" {
				t.Fatalf("first event=%q err=%v", line, err)
			}
			h := <-headers
			if h.Get("X-Agent-Account-ID") != "user-a" || h.Get("Authorization") != "" || h.Get("Cookie") != "" || h.Get("Forwarded") != "" {
				t.Fatal("proxy failed to replace identity or strip credentials/forwarded headers")
			}
			if disconnect {
				res.Body.Close()
				select {
				case <-canceled:
				case <-ctx.Done():
					t.Fatal("upstream did not observe client disconnect")
				}
			} else {
				close(release)
				rest, err := io.ReadAll(reader)
				if err != nil || !strings.Contains(string(rest), "event: done\n") {
					t.Fatalf("missing end event: %q err=%v", rest, err)
				}
			}
		})
	}
}

func TestAgentURLValidation(t *testing.T) {
	for _, raw := range []string{"invalid", "https://localhost:8082", "http://user:password@localhost", "http://localhost/path", "http://localhost?x=1", "http://localhost#fragment"} {
		if _, err := NewAgentProxy(raw); err == nil {
			t.Errorf("accepted invalid internal origin %q", raw)
		}
	}
}

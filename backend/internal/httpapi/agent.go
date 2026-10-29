package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// NewAgentProxy preserves the API path; only the configured internal host changes.
// An empty URL keeps the business API usable without a Node service.
func NewAgentProxy(rawURL string) (http.Handler, error) {
	if rawURL == "" {
		return nil, nil
	}
	target, err := url.Parse(rawURL)
	if err != nil || target.Scheme != "http" || target.Hostname() == "" || target.User != nil || target.Path != "" && target.Path != "/" || target.RawQuery != "" || target.Fragment != "" {
		return nil, fmt.Errorf("AGENT_INTERNAL_URL must be an http origin, e.g. http://127.0.0.1:8082")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Internal traffic must not go through a machine's HTTP proxy. Limit the
	// wait for headers, but allow the response body to remain a live stream.
	transport.Proxy = nil
	transport.ResponseHeaderTimeout = 5 * time.Second
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			account, _ := accountFromContext(pr.In.Context())
			pr.Out.Header.Del("Authorization")
			pr.Out.Header.Del("Cookie")
			pr.Out.Header.Set("X-Agent-Account-ID", account.AccountID)
		},
		Transport:     transport,
		FlushInterval: -1, // Flush every write; never buffer a complete SSE answer.
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, _ error) {
			if r.Context().Err() == nil {
				writeError(w, http.StatusBadGateway, "agent_unavailable", "Agent 服务暂不可用")
			}
		},
	}, nil
}

func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireUser(w, r); !ok {
		return
	}
	if s.agent == nil {
		writeError(w, http.StatusServiceUnavailable, "agent_not_configured", "尚未配置 Agent 服务")
		return
	}
	s.agent.ServeHTTP(w, r)
}

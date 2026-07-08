package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jpoz/mirra/internal/config"
	"github.com/jpoz/mirra/internal/recorder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Codex sends ChatGPT-subscription traffic to OpenAI-shaped paths with a
// ChatGPT-Account-ID header; those requests must reach the chatgpt upstream
// with the /v1 prefix dropped, while plain API-key traffic keeps going to the
// openai upstream unchanged.
func TestHandle_ChatGPTAccountHeaderSwitchesUpstream(t *testing.T) {
	requests := map[string]string{}
	upstream := func(name string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests[name] = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
	}
	openaiUpstream := upstream("openai")
	defer openaiUpstream.Close()
	chatgptUpstream := upstream("chatgpt")
	defer chatgptUpstream.Close()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"openai":  {UpstreamURL: openaiUpstream.URL},
			"chatgpt": {UpstreamURL: chatgptUpstream.URL + "/backend-api/codex"},
		},
	}
	p := New(cfg, recorder.New(false, ""))

	tests := []struct {
		name         string
		header       string
		wantUpstream string
		wantPath     string
	}{
		{
			name:         "chatgpt subscription auth",
			header:       "acct-123",
			wantUpstream: "chatgpt",
			wantPath:     "/backend-api/codex/responses",
		},
		{
			name:         "api key auth",
			wantUpstream: "openai",
			wantPath:     "/v1/responses",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clear(requests)

			req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5.5"}`))
			if tt.header != "" {
				req.Header.Set("ChatGPT-Account-ID", tt.header)
			}
			w := httptest.NewRecorder()
			p.Handle(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, map[string]string{tt.wantUpstream: tt.wantPath}, requests)
		})
	}
}

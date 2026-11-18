package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccessLogAllowedOrigin(t *testing.T) {
	const tailscaleURL = "http://100.118.125.84:30082"
	for _, tc := range []struct {
		name, configuredURL, requestURL, origin string
		want                                    int
	}{
		{"loopback", "", "http://127.0.0.1:8082", "http://127.0.0.1:8082", http.StatusNoContent},
		{"unconfigured remote", "", tailscaleURL, "", http.StatusForbidden},
		{"invalid configuration", tailscaleURL + "/path", tailscaleURL, "", http.StatusForbidden},
		{"configured API GET", tailscaleURL, tailscaleURL, "", http.StatusNoContent},
		{"configured same-origin request", tailscaleURL, tailscaleURL, tailscaleURL, http.StatusNoContent},
		{"different origin scheme", tailscaleURL, tailscaleURL, "https://100.118.125.84:30082", http.StatusForbidden},
		{"different origin host", tailscaleURL, tailscaleURL, "http://other.example", http.StatusForbidden},
		{"different request port", tailscaleURL, "http://100.118.125.84:8082", "", http.StatusForbidden},
		{"configured HTTPS", "https://skillhub.example/", "https://skillhub.example", "https://skillhub.example", http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SKILLHUB_URL", tc.configuredURL)
			handler := accessLogMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodGet, tc.requestURL+"/skillhub/v1/facets", nil)
			req.Header.Set("Origin", tc.origin)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != tc.want {
				t.Fatalf("status = %d, want %d, body = %q", res.Code, tc.want, res.Body.String())
			}
		})
	}
}

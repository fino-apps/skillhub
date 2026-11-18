package middleware

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/fino-io/finokit/logs"
	"github.com/fino-io/gokit/middleware/accesslog"
)

func accessLogMiddleware() Middleware {
	var publicURL *url.URL
	if configured := strings.TrimSpace(os.Getenv("SKILLHUB_URL")); configured != "" {
		u, err := url.Parse(strings.TrimSuffix(configured, "/"))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") ||
			u.Hostname() == "" || u.User != nil || u.Path != "" || u.ForceQuery || u.RawQuery != "" || u.Fragment != "" {
			logs.Warnw("SKILLHUB_URL must be an HTTP or HTTPS origin")
		} else {
			publicURL = u
		}
	}

	return func(next http.Handler) http.Handler {
		logged := accesslog.HTTPMiddleware()(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.Host)
			if err != nil {
				host = strings.Trim(r.Host, "[]")
			}
			scheme := "http"
			if publicURL != nil && strings.EqualFold(r.Host, publicURL.Host) {
				scheme = publicURL.Scheme
			} else if !strings.EqualFold(host, "localhost") && !net.ParseIP(host).IsLoopback() {
				http.Error(w, "SkillHub requires a loopback address", http.StatusForbidden)
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" && !strings.EqualFold(origin, scheme+"://"+r.Host) {
				http.Error(w, "SkillHub requires a same-origin request", http.StatusForbidden)
				return
			}
			w.Header().Set("X-Content-Type-Options", "nosniff")
			logged.ServeHTTP(w, r)
		})
	}
}

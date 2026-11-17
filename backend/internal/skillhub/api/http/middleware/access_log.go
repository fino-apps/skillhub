package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/fino-io/gokit/middleware/accesslog"
)

func accessLogMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		logged := accesslog.HTTPMiddleware()(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.Host)
			if err != nil {
				host = strings.Trim(r.Host, "[]")
			}
			if !strings.EqualFold(host, "localhost") && !net.ParseIP(host).IsLoopback() {
				http.Error(w, "SkillHub requires a loopback address", http.StatusForbidden)
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || u.Scheme != "http" || !strings.EqualFold(u.Host, r.Host) {
					http.Error(w, "SkillHub requires a same-origin request", http.StatusForbidden)
					return
				}
			}
			w.Header().Set("X-Content-Type-Options", "nosniff")
			logged.ServeHTTP(w, r)
		})
	}
}

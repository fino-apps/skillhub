package http

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// WithLocalRequests permits browser requests from the local SkillHub origin.
func WithLocalRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = strings.Trim(r.Host, "[]")
		}
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			writeAPIError(w, http.StatusForbidden, errors.New("SkillHub 仅接受本机请求"))
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			parsed, err := url.Parse(origin)
			if err != nil || parsed.Scheme != "http" || parsed.Host != r.Host {
				writeAPIError(w, http.StatusForbidden, errors.New("请通过 SkillHub 页面发起请求"))
				return
			}
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func writeAPIError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
}

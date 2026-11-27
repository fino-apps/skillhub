//go:build darwin && skillhub_desktop

package main

import (
	"crypto/subtle"
	"io/fs"
	"net/http"
	"strings"
	"sync/atomic"
)

func desktopHandler(host, launchPath, session string, assets fs.FS, api http.Handler) http.Handler {
	var launched atomic.Bool
	static := http.FileServerFS(assets)
	var crossOrigin http.CrossOriginProtection

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != host {
			http.Error(w, "Invalid host", http.StatusBadRequest)
			return
		}

		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.Method == http.MethodGet && r.URL.Path == launchPath && !launched.Swap(true) {
			w.Header().Set("Cache-Control", "no-store")
			http.SetCookie(w, &http.Cookie{
				Name: "skillhub_session", Value: session, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode,
			})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		cookie, err := r.Cookie("skillhub_session")
		if err != nil || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(session)) != 1 {
			http.Error(w, "Desktop session required", http.StatusUnauthorized)
			return
		}
		if err := crossOrigin.Check(r); err != nil {
			http.Error(w, "Cross-origin request denied", http.StatusForbidden)
			return
		}
		if r.URL.Path == "/skillhub/v1" || strings.HasPrefix(r.URL.Path, "/skillhub/v1/") {
			api.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path != "/" && strings.Contains(r.Header.Get("Accept"), "text/html") {
			if _, err := fs.Stat(assets, strings.TrimPrefix(r.URL.Path, "/")); err != nil {
				r = r.Clone(r.Context())
				r.URL.Path = "/"
			}
		}
		static.ServeHTTP(w, r)
	})
}

//go:build darwin && skillhub_desktop

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestDesktopHandler(t *testing.T) {
	const host = "127.0.0.1:12345"
	const launch = "/__launch/example"
	const apiPath = "/skillhub/v1/skills"
	handler := desktopHandler(host, launch, "session", fstest.MapFS{
		"index.html": {Data: []byte("desktop")},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))

	request := func(method, path, origin string, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "http://"+host+path, nil)
		req.Header.Set("Accept", "text/html")
		req.Header.Set("Origin", origin)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		return recorder
	}

	require.Equal(t, http.StatusUnauthorized, request("GET", apiPath, "", nil).Code)
	response := request("GET", launch, "", nil)
	require.Equal(t, http.StatusSeeOther, response.Code)
	require.Equal(t, "/", response.Header().Get("Location"))
	cookies := response.Result().Cookies()
	require.Len(t, cookies, 1)
	require.True(t, cookies[0].HttpOnly)
	require.Equal(t, http.SameSiteStrictMode, cookies[0].SameSite)
	require.Equal(t, http.StatusUnauthorized, request("GET", launch, "", nil).Code)
	require.Equal(t, http.StatusForbidden, request("POST", apiPath, "https://example.org", cookies[0]).Code)
	require.Equal(t, http.StatusNotFound, request("GET", apiPath, "", cookies[0]).Code)
	require.Equal(t, "desktop", request("GET", "/skills/example", "", cookies[0]).Body.String())
}

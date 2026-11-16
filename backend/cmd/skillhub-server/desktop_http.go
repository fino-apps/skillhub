package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/fino-io/skillhub/backend/internal/skillhub/app"
)

type archiveService interface {
	StageUpload(io.Reader) (app.UploadPreview, error)
	Archive(context.Context, string, string, io.Writer) (string, error)
}

func desktopHandler(api http.Handler, archives archiveService, assets fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/skillhub/", api)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("POST /skillhub/v1/uploads", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, (64<<20)+(1<<20))
		form, err := r.MultipartReader()
		if err != nil {
			writeDesktopError(w, http.StatusBadRequest, errors.New("请选择一个 ZIP 文件"))
			return
		}
		part, err := form.NextPart()
		if err != nil || part.FormName() != "file" || part.FileName() == "" {
			writeDesktopError(w, http.StatusBadRequest, errors.New("请选择一个 ZIP 文件"))
			return
		}
		defer part.Close()
		preview, err := archives.StageUpload(part)
		if err != nil {
			writeDesktopError(w, http.StatusBadRequest, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(preview)
	})
	mux.HandleFunc("GET /skillhub/v1/skills/{skill}/versions/{version}/archive", func(w http.ResponseWriter, r *http.Request) {
		file, err := os.CreateTemp("", "skillhub-archive-*.zip")
		if err != nil {
			writeDesktopError(w, http.StatusInternalServerError, err)
			return
		}
		defer os.Remove(file.Name())
		defer file.Close()
		name, err := archives.Archive(r.Context(), r.PathValue("skill"), r.PathValue("version"), file)
		if err != nil {
			writeDesktopError(w, http.StatusNotFound, err)
			return
		}
		info, err := file.Stat()
		if err != nil {
			writeDesktopError(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
		http.ServeContent(w, r, name, info.ModTime(), file)
	})
	if assets != nil {
		files := http.FileServerFS(assets)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if r.URL.Path == "/" || r.URL.Path == "/console" || r.URL.Path == "/issues" || strings.HasPrefix(r.URL.Path, "/skills/") {
				r.URL.Path = "/"
			}
			files.ServeHTTP(w, r)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = strings.Trim(r.Host, "[]")
		}
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			writeDesktopError(w, http.StatusForbidden, errors.New("SkillHub 仅接受本机请求"))
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			parsed, err := url.Parse(origin)
			if err != nil || parsed.Scheme != "http" || parsed.Host != r.Host {
				writeDesktopError(w, http.StatusForbidden, errors.New("请通过 SkillHub 页面发起请求"))
				return
			}
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mux.ServeHTTP(w, r)
	})
}

func writeDesktopError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
}

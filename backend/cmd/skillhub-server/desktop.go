//go:build darwin && skillhub_desktop

package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/config/source/memory"
	"github.com/fino-io/finokit/logs"
	gokitmetrics "github.com/fino-io/gokit/metrics"
	nserver "github.com/fino-io/gokit/server"
	"github.com/fino-io/skillhub/backend/internal/skillhub/api/endpoint"
	"github.com/fino-io/skillhub/backend/internal/skillhub/api/endpoint/middleware"
	skillhttp "github.com/fino-io/skillhub/backend/internal/skillhub/api/http"
	"go.opentelemetry.io/otel/trace/noop"
)

//go:embed all:desktop-assets
var desktopAssets embed.FS

func loadConfig() (nserver.Config, error) {
	data, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return nserver.Config{}, err
	}
	logDir, err := mygo.App.Path(mygo.PathLogs)
	if err != nil {
		return nserver.Config{}, err
	}
	dsn := url.URL{
		Scheme:   "file",
		Path:     filepath.Join(data, "catalog.db"),
		RawQuery: "_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on",
	}
	var paginationKey [32]byte
	rand.Read(paginationKey[:])
	settings, err := json.Marshal(map[string]any{
		"db":         map[string]string{"driver": "sqlite", "dsn": dsn.String()},
		"pagination": map[string]string{"encodedKey": base64.StdEncoding.EncodeToString(paginationKey[:])},
		"logs":       map[string]any{"level": "info", "output": "file", "file": map[string]string{"path": filepath.Join(logDir, "app.log")}},
	})
	if err != nil {
		return nserver.Config{}, err
	}
	if err := config.InitDefault(config.WithWatcherDisabled(), config.WithSource(memory.NewSource(memory.WithJSON(settings)))); err != nil {
		return nserver.Config{}, err
	}
	return nserver.Config{HttpAddr: "127.0.0.1:0"}, logs.NewConfig()
}

func serve(ctx context.Context, cfg nserver.Config) error {
	if !mygo.App.RequestSingleInstanceLock() {
		return nil
	}

	return runDesktop(ctx, cfg.HttpAddr)
}

func runDesktop(ctx context.Context, address string) (err error) {
	data, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return err
	}
	paths := libraryPaths{
		snapshots: filepath.Join(data, "snapshots"),
		exports:   filepath.Join(data, "exports"),
		uploads:   filepath.Join(data, "uploads"),
	}
	if err := paths.prepare(); err != nil {
		return err
	}

	deps, err := newInfra()
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, deps.close()) }()

	var window atomic.Pointer[mygo.Window]
	pickDirectory := func(ctx context.Context) (string, error) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: window.Load(), Title: "选择文件夹", Directory: true,
		})
		if err != nil || len(paths) == 0 {
			return "", err
		}
		return paths[0], nil
	}
	service, err := buildServer(deps, paths, pickDirectory)
	if err != nil {
		return err
	}

	assets, err := fs.Sub(desktopAssets, "desktop-assets")
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		return err
	}
	defer listener.Close()

	host := listener.Addr().String()
	logs.Infow("desktop HTTP listening", "address", host)
	origin := "http://" + host
	launchPath := "/__launch/" + rand.Text()
	tracer := noop.NewTracerProvider().Tracer("skillhub-desktop")
	endpoints := middleware.WrapEndpoints(endpoint.NewEndpoints(service), middleware.DefaultOptions(tracer))
	api := skillhttp.NewHandler(endpoints, tracer, gokitmetrics.Disabled().HTTPMiddleware)
	server := &http.Server{
		Handler:           desktopHandler(host, launchPath, rand.Text(), assets, api),
		ReadHeaderTimeout: 5 * time.Second,
	}
	stopped := make(chan error, 1)
	go func() {
		stopped <- server.Serve(listener)
	}()

	mygo.App.OnSecondInstance(func(_ []string, _ string) {
		if win := window.Load(); win != nil {
			win.Restore()
			win.Focus()
		}
	})
	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title: "SkillHub", Width: 1200, Height: 800, MinWidth: 720, MinHeight: 520,
			StateKey: "library", URL: "about:blank",
		})
		window.Store(win)
		attachDesktopNavigation(win, origin)
		if err := win.Page().LoadURL(origin + launchPath); err != nil {
			logs.Errorw("failed to load desktop", "error", err)
			mygo.Dialog.Error("SkillHub 启动失败", err.Error())
			mygo.App.Quit()
		}
	})
	stop := context.AfterFunc(ctx, mygo.App.Quit)
	defer stop()
	err = mygo.App.Run()

	deadline, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if shutdownErr := server.Shutdown(deadline); shutdownErr != nil {
		err = errors.Join(err, shutdownErr, server.Close())
	}
	if serveErr := <-stopped; !errors.Is(serveErr, http.ErrServerClosed) {
		err = errors.Join(err, serveErr)
	}
	return err
}

func attachDesktopNavigation(win *mygo.Window, origin string) {
	openExternal := func(rawURL string) {
		u, err := url.Parse(rawURL)
		if err == nil && u.Host != "" && (u.Scheme == "http" || u.Scheme == "https") {
			go func() {
				if err := mygo.Shell.OpenExternal(rawURL); err != nil {
					logs.Warnw("failed to open external link", "error", err)
				}
			}()
		}
	}
	win.Page().OnWillNavigate(func(event *mygo.NavigateEvent) {
		if strings.HasPrefix(event.URL, origin+"/") || strings.HasPrefix(event.URL, "blob:"+origin+"/") {
			return
		}
		event.PreventDefault()
		if event.UserInitiated {
			openExternal(event.URL)
		}
	})
	win.Page().SetWindowOpenHandler(func(request mygo.WindowOpenRequest) *mygo.WindowOptions {
		if request.UserInitiated {
			openExternal(request.URL)
		}
		return nil
	})
}

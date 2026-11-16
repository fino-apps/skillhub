package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/config/source/memory"
	"github.com/fino-io/finokit/db"
	"github.com/fino-io/finokit/pagination"
	gokitmetrics "github.com/fino-io/gokit/metrics"
	nserver "github.com/fino-io/gokit/server"
	"github.com/fino-io/skillhub/backend/internal/skillhub/api/endpoint"
	endpointmw "github.com/fino-io/skillhub/backend/internal/skillhub/api/endpoint/middleware"
	finohttp "github.com/fino-io/skillhub/backend/internal/skillhub/api/http"
	"github.com/fino-io/skillhub/backend/internal/skillhub/app"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	infrarepo "github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo/dao"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/otel/trace/noop"
)

func loadConfig() (nserver.Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nserver.Config{}, err
	}
	port := "31101"
	if value := os.Getenv("SERVER_HTTP_PORT"); value != "" {
		port = value
	}
	if number, err := strconv.Atoi(port); err != nil || number < 1 || number > 65535 {
		return nserver.Config{}, errors.New("SERVER_HTTP_PORT must be between 1 and 65535")
	}
	defaults := []byte(`{"transport":{"rateLimit":{"enabled":false},"accessLog":{},"http":{"cors":{"allowedOrigins":[]}}}}`)
	if err := config.InitDefault(config.WithWatcherDisabled(), config.WithSource(memory.NewSource(memory.WithJSON(defaults)))); err != nil {
		return nserver.Config{}, err
	}
	return nserver.Config{HttpAddr: net.JoinHostPort("127.0.0.1", port)}, nil
}

func newDesktop(ctx context.Context) (handler http.Handler, close func() error, err error) {
	data, err := dataDirectory()
	if err != nil {
		return nil, nil, err
	}
	database := os.Getenv("SKILLHUB_DATABASE")
	if database == "" {
		database = filepath.Join(data, "catalog.db")
	}
	database, err = filepath.Abs(database)
	if err != nil {
		return nil, nil, err
	}
	dsn := url.URL{Scheme: "file", Path: filepath.ToSlash(database), RawQuery: "_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"}
	provider, err := db.NewWithConfig(&db.Config{Driver: db.SqliteDriver, DSN: dsn.String(), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil {
			_ = provider.Close()
		}
	}()
	if err := migrateSchema(provider); err != nil {
		return nil, nil, err
	}
	if err := prepareCatalog(ctx, provider); err != nil {
		return nil, nil, err
	}
	exportDir, uploadDir := filepath.Join(data, "exports"), filepath.Join(data, "uploads")
	if configured := os.Getenv("SKILLHUB_EXPORT_DIR"); configured != "" {
		exportDir, err = filepath.Abs(configured)
		if err != nil {
			return nil, nil, err
		}
	}
	for _, dir := range []string{exportDir, uploadDir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, nil, err
		}
	}
	snapshotDir := os.Getenv("SKILLHUB_SNAPSHOT_DIR")
	if snapshotDir == "" {
		snapshotDir = filepath.Join(data, "snapshots")
	}
	snapshots, err := service.NewSnapshotStore(snapshotDir)
	if err != nil {
		return nil, nil, err
	}
	var paginationKey [32]byte
	if _, err := rand.Read(paginationKey[:]); err != nil {
		return nil, nil, err
	}
	pages, err := pagination.NewCursorCodecFromBase64(base64.StdEncoding.EncodeToString(paginationKey[:]))
	if err != nil {
		return nil, nil, err
	}
	catalog := infrarepo.NewCatalogRepo(provider)
	if err := snapshots.Prune(func(skillID, versionID string) (bool, error) {
		return catalog.HasVersion(ctx, skillID, versionID)
	}); err != nil {
		log.Print("snapshot cleanup failed: ", err)
	}
	svc := app.NewService(app.ServiceDeps{
		Catalog: catalog, Snapshots: snapshots, ExportDir: exportDir, UploadDir: uploadDir,
		AIModel: strings.TrimSpace(os.Getenv("SKILLHUB_OLLAMA_MODEL")),
		SkillhubSvc: service.NewSkillhubService(service.ServiceDeps{
			SkillRepo: infrarepo.NewSkillRepo(infrarepo.SkillRepoDeps{
				SkillDao: dao.NewSkillDaoImpl(provider), PageTokenCodec: pages,
			}),
			SkillVersionRepo: infrarepo.NewSkillVersionRepo(infrarepo.SkillVersionRepoDeps{
				SkillVersionDao: dao.NewSkillVersionDaoImpl(provider), PageTokenCodec: pages,
			}),
		}),
	})
	tracer := noop.NewTracerProvider().Tracer("skillhub")
	api := finohttp.NewHandler(endpointmw.WrapEndpoints(endpoint.NewEndpoints(svc), endpointmw.DefaultOptions(tracer)),
		tracer, gokitmetrics.Disabled().HTTPMiddleware)
	return desktopHandler(api, svc, frontendFiles), provider.Close, nil
}

func serve(ctx context.Context, cfg nserver.Config) error {
	listener, err := net.Listen("tcp", cfg.HttpAddr)
	if err != nil {
		return err
	}
	defer listener.Close()
	handler, close, err := newDesktop(ctx)
	if err != nil {
		return err
	}
	defer close()

	server := &http.Server{
		Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second,
	}
	log.Printf("SkillHub http://%s", listener.Addr())
	stopped := make(chan error, 1)
	go func() { stopped <- server.Serve(listener) }()
	select {
	case err := <-stopped:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}

func dataDirectory() (string, error) {
	dir := os.Getenv("SKILLHUB_DATA_DIR")
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(base, "SkillHub")
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(dir)
}

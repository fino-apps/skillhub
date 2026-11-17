package main

import (
	"context"

	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"
	gokitmetrics "github.com/fino-io/gokit/metrics"
	nserver "github.com/fino-io/gokit/server"
	"github.com/fino-io/gokit/tracing"
	"github.com/fino-io/skillhub/backend/infra/nacos"
	"github.com/fino-io/skillhub/backend/internal/skillhub/api"
	"github.com/fino-io/skillhub/backend/internal/skillhub/app"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	infrarepo "github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func loadConfig() (nserver.Config, error) {
	if err := nacos.LoadServiceConfig("skillhub-server"); err != nil {
		return nserver.Config{}, err
	}

	var cfg nserver.Config
	if err := config.ScanFrom(&cfg, "skillhubServer", "server"); err != nil {
		return nserver.Config{}, err
	}

	if err := logs.NewConfig(); err != nil {
		return nserver.Config{}, err
	}

	return cfg, nil
}

func serve(ctx context.Context, cfg nserver.Config) error {
	tracer, shutdownTrace, err := tracing.New("skillhub-server")
	if err != nil {
		logs.Warnw("failed to initialize tracing", "error", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), nserver.DefaultShutdownTimeout)
		defer cancel()
		if err := shutdownTrace(shutdownCtx); err != nil {
			logs.Warnw("failed to shut down tracing", "error", err)
		}
	}()

	instrumentation, err := gokitmetrics.New("skillhub")
	if err != nil {
		logs.Warnw("failed to initialize metrics", "error", err)
		instrumentation = gokitmetrics.Disabled()
	}

	paths, err := prepareLibraryPaths()
	if err != nil {
		return err
	}
	infra, err := newInfra()
	if err != nil {
		return err
	}
	defer func() {
		if err := infra.close(); err != nil {
			logs.Errorw("failed to close infrastructure", "error", err)
		}
	}()

	server, err := buildServer(infra, paths)
	if err != nil {
		return err
	}

	return api.RunContext(ctx, cfg, server, tracer, instrumentation)
}

func buildServer(deps infra, paths libraryPaths) (pb.SkillhubServiceServer, error) {
	if err := migrateSchema(deps.DB); err != nil {
		return nil, err
	}

	libraryRepo := infrarepo.NewLibrary(infrarepo.LibraryDeps{
		DB:             deps.DB,
		PageTokenCodec: deps.Pagination,
	})

	libraryService := service.NewLibrary(service.LibraryDeps{
		Store:       libraryRepo,
		SnapshotDir: paths.snapshots,
		UploadDir:   paths.uploads,
		ExportDir:   paths.exports,
	})

	return app.NewService(app.ServiceDeps{
		Library: libraryService,
	}), nil
}

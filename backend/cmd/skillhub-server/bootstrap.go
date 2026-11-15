package main

import (
	"context"
	"os"

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
	"github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo/dao"
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

	infra, err := newInfra()
	if err != nil {
		return err
	}
	defer func() {
		if err := infra.close(); err != nil {
			logs.Errorw("failed to close infrastructure", "error", err)
		}
	}()

	server, err := buildServer(infra)
	if err != nil {
		return err
	}

	return api.RunContext(ctx, cfg, server, tracer, instrumentation)
}

func buildServer(deps infra) (pb.SkillhubServiceServer, error) {
	if err := migrateSchema(deps.DB); err != nil {
		return nil, err
	}

	// The current indexes also distinguish independent local snapshots.
	session := deps.DB.NewSession(context.Background())
	for _, index := range []string{"idx_skill_source", "idx_skill_commit"} {
		if err := session.Exec("DROP INDEX IF EXISTS " + index).Error; err != nil {
			return nil, err
		}
	}
	snapshotDir := os.Getenv("SKILLHUB_SNAPSHOT_DIR")
	if snapshotDir == "" {
		snapshotDir = "skillhub-snapshots"
	}
	snapshots, err := service.NewSnapshotStore(snapshotDir)
	if err != nil {
		return nil, err
	}

	skillRepo := infrarepo.NewSkillRepo(infrarepo.SkillRepoDeps{
		SkillDao:       dao.NewSkillDaoImpl(deps.DB),
		PageTokenCodec: deps.Pagination,
	})
	skillVersionRepo := infrarepo.NewSkillVersionRepo(infrarepo.SkillVersionRepoDeps{
		SkillVersionDao: dao.NewSkillVersionDaoImpl(deps.DB),
		PageTokenCodec:  deps.Pagination,
	})

	skillhubService := service.NewSkillhubService(service.ServiceDeps{

		SkillRepo:        skillRepo,
		SkillVersionRepo: skillVersionRepo,
	})

	return app.NewService(app.ServiceDeps{
		Catalog:     infrarepo.NewCatalogRepo(deps.DB),
		Snapshots:   snapshots,
		SkillhubSvc: skillhubService,
	}), nil
}

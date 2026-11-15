package main

import (
	"context"
	"os"
	"path/filepath"

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

	projectSkillRepo := infrarepo.NewProjectSkillRepo(infrarepo.ProjectSkillRepoDeps{
		ProjectSkillDao: dao.NewProjectSkillDaoImpl(deps.DB),
		PageTokenCodec:  deps.Pagination,
	})
	skillRepo := infrarepo.NewSkillRepo(infrarepo.SkillRepoDeps{
		SkillDao:       dao.NewSkillDaoImpl(deps.DB),
		PageTokenCodec: deps.Pagination,
	})
	skillTagRepo := infrarepo.NewSkillTagRepo(infrarepo.SkillTagRepoDeps{
		SkillTagDao:    dao.NewSkillTagDaoImpl(deps.DB),
		PageTokenCodec: deps.Pagination,
	})
	skillVersionRepo := infrarepo.NewSkillVersionRepo(infrarepo.SkillVersionRepoDeps{
		SkillVersionDao: dao.NewSkillVersionDaoImpl(deps.DB),
		PageTokenCodec:  deps.Pagination,
	})

	skillhubService := service.NewSkillhubService(service.ServiceDeps{
		ProjectSkillRepo: projectSkillRepo,
		SkillRepo:        skillRepo,
		SkillTagRepo:     skillTagRepo,
		SkillVersionRepo: skillVersionRepo,
	})

	dataRoot := os.Getenv("SKILLHUB_DATA_DIR")
	if dataRoot == "" {
		dataRoot = "skillhub-data"
	}
	dataRoot, err := filepath.Abs(dataRoot)
	if err != nil {
		return nil, err
	}

	return app.NewService(app.ServiceDeps{
		Intake:      service.NewIntake(dataRoot, infrarepo.NewIntakeRepo(deps.DB)),
		SkillhubSvc: skillhubService,
	}), nil
}

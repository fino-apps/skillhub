package main

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"

	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/config/source/memory"
	"github.com/fino-io/finokit/db"
	"github.com/fino-io/finokit/logs"
	"github.com/fino-io/skillhub/backend/internal/skillhub/app"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	infrarepo "github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo/dao"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

// openCatalog adds SkillHub's storage defaults and assembly to the generated infrastructure.
func openCatalog(ctx context.Context) (pb.SkillhubServiceServer, func() error, error) {
	data, err := dataDirectory()
	if err != nil {
		return nil, nil, err
	}
	if err := configureCatalog(data); err != nil {
		return nil, nil, err
	}
	deps, err := newInfra()
	if err != nil {
		return nil, nil, err
	}
	server, err := buildServer(ctx, deps, data)
	if err != nil {
		return nil, nil, errors.Join(err, deps.close())
	}
	return server, deps.close, nil
}

func configureCatalog(data string) error {
	overrides := make(map[string]any)
	var databaseConfig db.Config
	if err := config.ScanFrom(&databaseConfig, "db"); err != nil {
		return err
	}
	// The old bootstrap ignored the bundled DSN and stored the catalog in the data directory.
	// Continue opening that catalog while honoring a separately configured DSN.
	const bundledDSN = "file:skillhub-catalog.db?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"
	databaseOptions := make(map[string]any)
	database := os.Getenv("SKILLHUB_DATABASE")
	if database != "" || databaseConfig.DSN == "" || databaseConfig.DSN == bundledDSN {
		if database == "" {
			database = filepath.Join(data, "catalog.db")
		}
		database, err := filepath.Abs(database)
		if err != nil {
			return err
		}
		file := url.URL{Scheme: "file", Path: filepath.ToSlash(database), RawQuery: "_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"}
		databaseOptions["dsn"] = file.String()
	}
	if databaseConfig.Driver == db.SqliteDriver {
		if databaseConfig.MaxOpenConns == 0 {
			databaseOptions["maxOpenConns"] = 1
		}
		if databaseConfig.MaxIdleConns == 0 {
			databaseOptions["maxIdleConns"] = 1
		}
	}
	if len(databaseOptions) != 0 {
		overrides["db"] = databaseOptions
	}

	var key string
	if err := config.ScanFrom(&key, "pagination.encodedKey"); err != nil {
		return err
	}
	if key == "" {
		var random [32]byte
		if _, err := rand.Read(random[:]); err != nil {
			return err
		}
		overrides["pagination"] = map[string]string{"encodedKey": base64.StdEncoding.EncodeToString(random[:])}
	}
	if len(overrides) == 0 {
		return nil
	}
	raw, err := json.Marshal(overrides)
	if err != nil {
		return err
	}
	return config.Load(memory.NewSource(memory.WithJSON(raw)))
}

func buildServer(ctx context.Context, deps infra, data string) (pb.SkillhubServiceServer, error) {
	if err := migrateSchema(deps.DB); err != nil {
		return nil, err
	}
	if err := prepareCatalog(ctx, deps.DB); err != nil {
		return nil, err
	}
	exportDir, err := filepath.Abs(cmp.Or(os.Getenv("SKILLHUB_EXPORT_DIR"), filepath.Join(data, "exports")))
	if err != nil {
		return nil, err
	}
	uploadDir := filepath.Join(data, "uploads")
	for _, dir := range []string{exportDir, uploadDir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
	}
	snapshotDir := cmp.Or(os.Getenv("SKILLHUB_SNAPSHOT_DIR"), filepath.Join(data, "snapshots"))
	snapshots, err := service.NewSnapshotStore(snapshotDir)
	if err != nil {
		return nil, err
	}
	catalog := infrarepo.NewCatalogRepo(deps.DB)
	if err := snapshots.Prune(func(skillID, versionID string) (bool, error) {
		return catalog.HasVersion(ctx, skillID, versionID)
	}); err != nil {
		logs.Warnw("snapshot cleanup failed", "error", err)
	}
	svc := app.NewService(app.ServiceDeps{
		Catalog: catalog, Snapshots: snapshots, ExportDir: exportDir, UploadDir: uploadDir,
		SkillhubSvc: service.NewSkillhubService(service.ServiceDeps{
			SkillRepo: infrarepo.NewSkillRepo(infrarepo.SkillRepoDeps{
				SkillDao: dao.NewSkillDaoImpl(deps.DB), PageTokenCodec: deps.Pagination,
			}),
			SkillVersionRepo: infrarepo.NewSkillVersionRepo(infrarepo.SkillVersionRepoDeps{
				SkillVersionDao: dao.NewSkillVersionDaoImpl(deps.DB), PageTokenCodec: deps.Pagination,
			}),
		}),
	})
	return svc, nil
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

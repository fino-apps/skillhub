package main

import (
	"context"

	"github.com/fino-io/skillhub/backend/internal/skillhub/app"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	infrarepo "github.com/fino-io/skillhub/backend/internal/skillhub/infra/repo"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func buildServer(deps infra, paths libraryPaths, pickDirectory func(context.Context) (string, error)) (pb.SkillhubServiceServer, error) {
	if err := migrateSchema(deps.DB); err != nil {
		return nil, err
	}

	libraryRepo := infrarepo.NewLibrary(infrarepo.LibraryDeps{
		DB:             deps.DB,
		PageTokenCodec: deps.Pagination,
	})

	libraryService := service.NewLibrary(service.LibraryDeps{
		Store:         libraryRepo,
		SnapshotDir:   paths.snapshots,
		UploadDir:     paths.uploads,
		ExportDir:     paths.exports,
		PickDirectory: pickDirectory,
	})

	return app.NewService(app.ServiceDeps{
		Library: libraryService,
	}), nil
}

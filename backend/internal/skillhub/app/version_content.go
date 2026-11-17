package app

import (
	"context"

	model "github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

func (s skillhubServer) versionView(row model.SkillVersionSnapshot) *pb.VersionView {
	return &pb.VersionView{
		Id: string(row.ID), SkillId: row.SkillId, Name: row.Name, Notes: row.Notes,
		SourceRef: row.SourceRef, CommitSha: row.CommitSha, ContentDigest: row.ContentDigest,
		Available: s.library.Available(row), CreateTime: row.CreateTime,
		UpstreamCommit: row.UpstreamCommit, UpstreamRepository: row.UpstreamRepository, UpstreamPath: row.UpstreamPath,
	}
}

func (s skillhubServer) queryVersions(ctx context.Context, in *pb.ListVersionsRequest) (*pb.ListVersionsResponse, error) {
	versions, token, err := s.library.Versions(ctx, in.GetSkillId(), int(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		return nil, failure(err)
	}
	result := &pb.ListVersionsResponse{NextPageToken: token}
	for _, version := range versions {
		result.Versions = append(result.Versions, s.versionView(version))
	}
	return result, nil
}

func (s skillhubServer) download(ctx context.Context, skill, id string) (*pb.DownloadZipResponse, error) {
	row, err := s.library.Version(ctx, skill, id)
	if err != nil {
		return nil, failure(err)
	}
	data, err := s.library.ReadArchive(row)
	if err != nil {
		return nil, failure(err)
	}
	return &pb.DownloadZipResponse{
		Archive:  &httpbody.HttpBody{Data: data, ContentType: "application/zip"},
		Filename: service.SafeName(row.Name) + ".zip",
	}, nil
}

func (s skillhubServer) versionContents(ctx context.Context, skill, version string) (*pb.GetVersionContentResponse, error) {
	result, err := s.library.VersionContents(ctx, skill, version)
	if err != nil {
		return nil, failure(err)
	}
	return &pb.GetVersionContentResponse{Description: result.Description, Files: result.Files}, nil
}

func (s skillhubServer) versionFile(ctx context.Context, skill, version, path string) (*pb.GetVersionFileResponse, error) {
	result, err := s.library.VersionFile(ctx, skill, version, path)
	if err != nil {
		return nil, failure(err)
	}
	return &pb.GetVersionFileResponse{Text: result.Text, Size: result.Size, Binary: result.Binary, TooLarge: result.TooLarge}, nil
}

func (s skillhubServer) exportVersion(ctx context.Context, in *pb.ExportVersionToDirectoryRequest) (*pb.ExportVersionToDirectoryResponse, error) {
	path, err := s.library.Export(ctx, in.GetSkillId(), in.GetVersionId(), in.GetDirectory(), in.GetChooseDirectory())
	if err != nil {
		return nil, failure(err)
	}
	return &pb.ExportVersionToDirectoryResponse{Path: path}, nil
}

func (s skillhubServer) installation(ctx context.Context, in *pb.GetInstallCommandRequest) (*pb.GetInstallCommandResponse, error) {
	command, err := s.library.InstallCommand(ctx, in.GetSkillId(), in.GetVersionId(), in.GetGlobal())
	if err != nil {
		return nil, failure(err)
	}
	return &pb.GetInstallCommandResponse{Command: command}, nil
}

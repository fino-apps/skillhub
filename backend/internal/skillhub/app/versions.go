package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/finokit/logs"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
)

func (s skillhubServer) versionView(version *model.SkillVersion) *pb.VersionView {
	_, err := s.versionPath(version)
	return &pb.VersionView{
		Id: string(version.ID()), SkillId: version.SkillId(), Name: version.Name(), Notes: version.Notes(),
		SourceRef: version.SourceRef(), CommitSha: version.CommitSha(), ContentDigest: version.ContentDigest(),
		UpstreamCommit: version.UpstreamCommit(), UpstreamRepository: version.UpstreamRepository(), UpstreamPath: version.UpstreamPath(), CreateTime: version.CreateTime(), Available: err == nil,
	}
}

func (s skillhubServer) versionPath(version *model.SkillVersion) (string, error) {
	if s.snapshots == nil {
		return "", errors.New("version snapshot unavailable")
	}
	return s.snapshots.Path(version.SkillId(), string(version.ID()), version.Name())
}

func (s skillhubServer) ownedVersion(ctx context.Context, skillID, versionID string) (*model.SkillVersion, error) {
	version, err := s.skillhubSvc.GetSkillVersion(ctx, model.SkillVersionID(versionID))
	if err != nil {
		return nil, catalogError(err)
	}
	if version.SkillId() != skillID {
		return nil, errcode.NewPlatformNotFound()
	}
	return version, nil
}

func (s skillhubServer) listVersions(ctx context.Context, in *pb.ListVersionsRequest) (*pb.ListVersionsResponse, error) {
	result, err := s.skillhubSvc.ListSkillVersions(ctx, model.ListSkillVersionQuery{
		Filter: model.SkillVersionFilter{SkillID: in.GetSkillId()}, PageSize: int(in.GetPageSize()),
		PageToken: in.GetPageToken(), SkipTotalCount: true,
	})
	if err != nil {
		return nil, catalogError(err)
	}
	items := make([]*pb.VersionView, 0, len(result.Items))
	for _, version := range result.Items {
		items = append(items, s.versionView(version))
	}
	return &pb.ListVersionsResponse{Versions: items, NextPageToken: result.NextPageToken}, nil
}

func (s skillhubServer) createVersion(ctx context.Context, in *pb.CreateVersionRequest) (*pb.CreateVersionResponse, error) {
	skill, err := s.skillhubSvc.GetSkill(ctx, model.SkillID(in.GetSkillId()))
	if err != nil {
		return nil, catalogError(err)
	}
	input := in.GetVersion()
	if input == nil {
		return nil, errcode.NewPlatformBadRequest()
	}
	if input.GetLocalPath() != "" || skill.RepositoryUrl() == "" {
		path := input.GetLocalPath()
		if path == "" {
			path = skill.LocalPath()
		}
		return s.saveNewVersion(ctx, skill, path, intakeSource{}, input)
	}
	if !service.IsCommit(input.GetCommitSha()) {
		return nil, errcode.NewPlatformBadRequest()
	}
	path, commit, cleanup, err := service.CheckoutGitHub(ctx, service.GitHubSource{
		RepositoryURL: skill.RepositoryUrl(), SkillPath: skill.RepositoryPath(), Ref: input.GetCommitSha(),
	})
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	defer cleanup()
	return s.saveNewVersion(ctx, skill, path, intakeSource{ref: input.GetSourceRef(), commit: commit}, input)
}

func (s skillhubServer) setDefaultVersion(ctx context.Context, in *pb.SetDefaultVersionRequest) (*core.Null, error) {
	s.files.Lock()
	defer s.files.Unlock()
	version, err := s.ownedVersion(ctx, in.GetSkillId(), in.GetVersionId())
	if err != nil {
		return nil, err
	}
	if _, err := s.versionPath(version); err != nil {
		return nil, errcode.WrapPlatformConflict(err)
	}
	if err := s.catalog.SetDefaultVersion(ctx, version); err != nil {
		return nil, catalogError(err)
	}
	return &core.Null{}, nil
}

func (s skillhubServer) deleteVersion(ctx context.Context, in *pb.DeleteVersionRequest) (*core.Null, error) {
	s.files.Lock()
	defer s.files.Unlock()
	version, err := s.ownedVersion(ctx, in.GetSkillId(), in.GetVersionId())
	if err != nil {
		return nil, err
	}
	if err := s.catalog.DeleteVersion(ctx, version); err != nil {
		return nil, catalogError(err)
	}
	s.removeSnapshot(in.GetSkillId(), in.GetVersionId())
	return &core.Null{}, nil
}

func (s skillhubServer) removeSnapshot(skillID, versionID string) {
	if err := s.snapshots.Remove(skillID, versionID); err != nil {
		logs.Warnw("failed to remove snapshot", "skill_id", skillID, "version_id", versionID, "error", err)
	}
}

func (s skillhubServer) getVersionContent(ctx context.Context, in *pb.GetVersionContentRequest) (*pb.GetVersionContentResponse, error) {
	s.files.RLock()
	defer s.files.RUnlock()
	version, err := s.ownedVersion(ctx, in.GetSkillId(), in.GetVersionId())
	if err != nil {
		return nil, err
	}
	path, err := s.versionPath(version)
	if err != nil {
		return nil, errcode.WrapPlatformNotFound(err)
	}
	files, err := service.SnapshotFiles(path)
	if err != nil {
		return nil, catalogError(err)
	}
	return &pb.GetVersionContentResponse{Description: version.Description(), Files: files}, nil
}

func (s skillhubServer) getVersionFile(ctx context.Context, in *pb.GetVersionFileRequest) (*pb.GetVersionFileResponse, error) {
	s.files.RLock()
	defer s.files.RUnlock()
	version, err := s.ownedVersion(ctx, in.GetSkillId(), in.GetVersionId())
	if err != nil {
		return nil, err
	}
	path, err := s.versionPath(version)
	if err != nil {
		return nil, errcode.WrapPlatformNotFound(err)
	}
	file, err := service.PreviewFile(path, in.GetPath())
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	return &pb.GetVersionFileResponse{Text: file.Text, Size: int32(file.Size), Binary: file.Binary, TooLarge: file.TooLarge}, nil
}

func (s skillhubServer) exportVersionToDirectory(ctx context.Context, in *pb.ExportVersionToDirectoryRequest) (*pb.ExportVersionToDirectoryResponse, error) {
	dir := in.GetDirectory()
	if in.GetChooseDirectory() {
		var err error
		dir, err = service.SelectDirectory(ctx)
		if err != nil {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
		if dir == "" {
			return &pb.ExportVersionToDirectoryResponse{}, nil
		}
	}
	s.files.Lock()
	defer s.files.Unlock()
	version, err := s.ownedVersion(ctx, in.GetSkillId(), in.GetVersionId())
	if err != nil {
		return nil, err
	}
	path, err := s.versionPath(version)
	if err != nil {
		return nil, errcode.WrapPlatformNotFound(err)
	}
	if dir == "" {
		dir = s.exportDir
		if home, homeErr := os.UserHomeDir(); homeErr == nil && os.Getenv("SKILLHUB_EXPORT_DIR") == "" {
			parent := filepath.Join(home, "Downloads")
			if _, statErr := os.Stat(parent); statErr == nil {
				target := filepath.Join(parent, "SkillHub")
				if os.MkdirAll(target, 0700) == nil {
					dir = target
				}
			}
		}
	}
	label := version.ContentDigest()
	if version.CommitSha() != "" {
		label = version.CommitSha()
	}
	label = label[:min(8, len(label))]
	saved, err := service.ExportDirectory(path, dir, version.Name(), label)
	if in.GetDirectory() == "" && !in.GetChooseDirectory() && dir != s.exportDir &&
		(errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist)) {
		saved, err = service.ExportDirectory(path, s.exportDir, version.Name(), label)
	}
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	return &pb.ExportVersionToDirectoryResponse{Path: saved}, nil
}

func (s skillhubServer) repairVersion(ctx context.Context, in *pb.RepairVersionRequest) (*core.Null, error) {
	version, err := s.ownedVersion(ctx, in.GetSkillId(), in.GetVersionId())
	if err != nil {
		return nil, err
	}
	if _, err := s.versionPath(version); err == nil {
		return &core.Null{}, nil
	}
	skill, err := s.skillhubSvc.GetSkill(ctx, model.SkillID(in.GetSkillId()))
	if err != nil {
		return nil, catalogError(err)
	}
	path, expectedDigest := skill.LocalPath(), version.ContentDigest()
	if version.CommitSha() != "" {
		var cleanup func()
		path, _, cleanup, err = service.CheckoutGitHub(ctx, service.GitHubSource{
			RepositoryURL: skill.RepositoryUrl(), SkillPath: skill.RepositoryPath(), Ref: version.CommitSha(),
		})
		if err != nil {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
		defer cleanup()
	}
	snapshot, err := s.snapshots.Capture(path, expectedDigest)
	if err != nil {
		return nil, errcode.WrapPlatformConflict(err)
	}
	keep := false
	defer func() {
		if !keep {
			snapshot.Discard()
		}
	}()
	if snapshot.Name != version.Name() {
		return nil, errcode.NewPlatformConflict()
	}
	s.files.Lock()
	defer s.files.Unlock()
	if _, err := s.ownedVersion(ctx, in.GetSkillId(), in.GetVersionId()); err != nil {
		return nil, err
	}
	if _, err := s.versionPath(version); err == nil {
		return &core.Null{}, nil
	}
	if err := s.snapshots.Publish(snapshot, in.GetSkillId(), in.GetVersionId()); err != nil {
		return nil, catalogError(err)
	}
	if err := s.catalog.UpdateSnapshot(ctx, version, snapshot.Digest, snapshot.Description); err != nil {
		return nil, catalogError(err)
	}
	keep = true
	return &core.Null{}, nil
}

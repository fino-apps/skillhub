package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
	"github.com/google/uuid"
)

func githubSource(source *pb.GitHubSource) service.GitHubSource {
	return service.GitHubSource{RepositoryURL: source.GetRepositoryUrl(), SkillPath: source.GetSkillPath(), Ref: source.GetRef()}
}

func candidates(items []service.ScannedSkill) []*pb.SkillCandidate {
	result := make([]*pb.SkillCandidate, 0, len(items))
	for _, item := range items {
		result = append(result, &pb.SkillCandidate{Path: item.Path, Name: item.Name, Description: item.Description})
	}
	return result
}

func (s skillhubServer) previewImport(ctx context.Context, in *pb.PreviewImportRequest) (*pb.PreviewImportResponse, error) {
	items, commit, err := service.InspectGitHub(ctx, githubSource(in.GetSource()), in.GetScanCollection())
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	return &pb.PreviewImportResponse{ResolvedCommitSha: commit, Skills: candidates(items)}, nil
}

type intakeSource struct {
	kind, repository, path, ref, commit       string
	upstreamName, upstreamURL, upstreamCommit string
	upstreamRepository, upstreamPath          string
	key                                       string
}

func sourceKey(parts ...string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(parts, "\x00"))))
}

func (s skillhubServer) importSkill(ctx context.Context, in *pb.ImportSkillRequest) (*pb.ImportSkillResponse, error) {
	if len(in.GetSkillPaths()) == 0 || len(in.GetSkillPaths()) > 100 {
		return nil, errcode.NewPlatformBadRequest()
	}
	reply := &pb.ImportSkillResponse{}
	if in.GetRepositoryUrl() == "" {
		for _, path := range in.GetSkillPaths() {
			added, err := s.saveImport(ctx, path, intakeSource{kind: "folder"})
			recordImport(reply, path, added, err)
		}
		return reply, nil
	}
	repository, err := service.NormalizeRepository(in.GetRepositoryUrl())
	if err != nil || !service.IsCommit(in.GetCommitSha()) {
		return nil, errcode.NewPlatformBadRequest()
	}
	root, commit, cleanup, err := service.CheckoutGitHub(ctx, service.GitHubSource{RepositoryURL: repository, Ref: in.GetCommitSha()})
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	defer cleanup()
	owner := strings.Split(strings.TrimPrefix(repository, "https://github.com/"), "/")[0]
	for _, path := range in.GetSkillPaths() {
		item, readErr := service.ReadGitHubSkill(root, path)
		if readErr != nil {
			recordImport(reply, path, false, readErr)
			continue
		}
		source := intakeSource{kind: "github", repository: repository, path: item.Path, ref: in.GetSourceRef(), commit: commit,
			upstreamName: owner, upstreamURL: repository, key: sourceKey("github", repository, item.Path)}
		added, saveErr := s.saveImport(ctx, filepath.Join(root, item.Path), source)
		recordImport(reply, path, added, saveErr)
	}
	return reply, nil
}

func recordImport(reply *pb.ImportSkillResponse, path string, added bool, err error) {
	switch {
	case err != nil:
		reply.Failures = append(reply.Failures, &pb.ImportFailure{Path: path, Message: err.Error()})
	case added:
		reply.AddedCount++
	default:
		reply.ExistingCount++
	}
}

func (s skillhubServer) saveImport(ctx context.Context, path string, source intakeSource) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	snapshot, err := s.snapshots.Capture(path, "")
	if err != nil {
		return false, err
	}
	localPath := ""
	if source.kind == "folder" {
		localPath = snapshot.Path
		source.key = sourceKey("folder", localPath)
	}
	if source.key == "" {
		source.key = sourceKey(source.kind, uuid.NewString())
	}
	skill, err := model.NewSkill(model.CreateSkillInput{
		Name: snapshot.Name, Description: snapshot.Description, ImportKind: source.kind, SourceKey: source.key,
		LocalPath: localPath, RepositoryUrl: source.repository, RepositoryPath: source.path,
		UpstreamName: source.upstreamName, UpstreamUrl: source.upstreamURL,
	})
	if err != nil {
		snapshot.Discard()
		return false, err
	}
	added := false
	defer func() {
		if !added {
			snapshot.Discard()
		}
	}()
	version, err := newVersion(skill.ID(), snapshot, source, "")
	if err != nil {
		return false, err
	}
	s.files.Lock()
	defer s.files.Unlock()
	if err := s.snapshots.Publish(snapshot, string(skill.ID()), string(version.ID())); err != nil {
		return false, err
	}
	added, err = s.catalog.AddSkill(ctx, repo.CatalogEntry{Skill: skill, Version: version})
	if err != nil {
		added = false
	}
	return added, err
}

func newVersion(id model.SkillID, snapshot *service.SkillSnapshot, source intakeSource, notes string) (*model.SkillVersion, error) {
	return model.NewSkillVersion(model.CreateSkillVersionInput{
		SkillId: string(id), Name: snapshot.Name, Description: snapshot.Description, Notes: notes,
		ContentDigest: snapshot.Digest, SourceRef: source.ref, CommitSha: source.commit, UpstreamCommit: source.upstreamCommit,
		UpstreamRepository: source.upstreamRepository, UpstreamPath: source.upstreamPath,
	})
}

func (s skillhubServer) saveNewVersion(ctx context.Context, skill *model.Skill, path string, source intakeSource, in *pb.VersionInput) (*pb.CreateVersionResponse, error) {
	snapshot, err := s.snapshots.Capture(path, in.GetExpectedDigest())
	if errors.Is(err, service.ErrSnapshotChanged) {
		return nil, errcode.WrapPlatformConflict(err)
	}
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	keep := false
	defer func() {
		if !keep {
			snapshot.Discard()
		}
	}()
	if source.commit == "" && snapshot.Name != skill.Name() {
		return nil, errcode.WrapPlatformBadRequest(errors.New("文件夹中的 Skill 名称与目标条目不同"))
	}
	version, err := newVersion(skill.ID(), snapshot, source, in.GetNotes())
	if err != nil {
		return nil, catalogError(err)
	}
	s.files.Lock()
	defer s.files.Unlock()
	if err := s.snapshots.Publish(snapshot, string(skill.ID()), string(version.ID())); err != nil {
		return nil, catalogError(err)
	}
	saved, err := s.catalog.AddVersion(ctx, version, in.GetMakeDefault())
	if err != nil {
		return nil, catalogError(err)
	}
	keep = saved.ID() == version.ID()
	if !keep && in.GetMakeDefault() {
		if _, err := s.versionPath(saved); err != nil {
			return nil, errcode.WrapPlatformConflict(errors.New("请先补齐已收录版本"))
		}
		if err := s.catalog.SetDefaultVersion(ctx, saved); err != nil {
			return nil, catalogError(err)
		}
	}
	return &pb.CreateVersionResponse{Version: s.versionView(saved)}, nil
}

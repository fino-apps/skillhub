package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fino-io/finokit/errorx"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/service"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	pb "github.com/fino-io/skillhub/go/skillhub/v1"
	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

func (s skillhubServer) uploadZip(file []byte) (*pb.UploadZipResponse, error) {
	preview, err := s.stageUpload(file)
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err, errorx.WithExtraMessage(err.Error()))
	}
	return preview, nil
}

func (s skillhubServer) stageUpload(file []byte) (*pb.UploadZipResponse, error) {
	if len(file) > 64<<20 {
		return nil, errors.New("ZIP 最大为 64 MiB")
	}
	s.uploads.Lock()
	defer s.uploads.Unlock()
	entries, err := os.ReadDir(s.uploadDir)
	if err != nil {
		return nil, err
	}
	active := 0
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if time.Since(info.ModTime()) > 30*time.Minute {
			_ = os.RemoveAll(filepath.Join(s.uploadDir, entry.Name()))
		} else {
			active++
		}
	}
	if active >= 8 {
		return nil, errors.New("当前暂存的 ZIP 过多，请稍后再试")
	}
	id := uuid.NewString()
	session := filepath.Join(s.uploadDir, id)
	if err := os.Mkdir(session, 0700); err != nil {
		return nil, err
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(session)
		}
	}()

	zipPath := filepath.Join(session, "upload.zip")
	if err := os.WriteFile(zipPath, file, 0600); err != nil {
		return nil, err
	}
	root := filepath.Join(session, "files")
	if err := os.Mkdir(root, 0700); err != nil {
		return nil, err
	}
	if err := service.ExtractArchive(zipPath, root); err != nil {
		return nil, err
	}
	_ = os.Remove(zipPath)
	items, err := service.ArchiveCandidates(root)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("ZIP 中没有可收录的 Skill")
	}
	keep = true
	return &pb.UploadZipResponse{UploadId: id, Skills: candidates(items)}, nil
}

func (s skillhubServer) commitUpload(ctx context.Context, in *pb.CommitUploadRequest) (*pb.ImportSkillResponse, error) {
	if uuid.Validate(in.GetUploadId()) != nil || len(in.GetSkillPaths()) == 0 {
		return nil, errcode.NewPlatformBadRequest()
	}
	s.uploads.Lock()
	defer s.uploads.Unlock()
	session := filepath.Join(s.uploadDir, in.GetUploadId())
	info, err := os.Stat(session)
	if err != nil || time.Since(info.ModTime()) > 30*time.Minute {
		return nil, errcode.NewPlatformNotFound()
	}
	root := filepath.Join(session, "files")
	items, err := service.ArchiveCandidates(root)
	if err != nil {
		return nil, errcode.WrapPlatformBadRequest(err)
	}
	allowed := make(map[string]bool, len(items))
	for _, item := range items {
		allowed[item.Path] = true
	}
	reply := &pb.ImportSkillResponse{}
	for _, relative := range in.GetSkillPaths() {
		if !allowed[relative] {
			return nil, errcode.NewPlatformBadRequest()
		}
	}
	origin := readArchiveOrigin(root)
	source := intakeSource{kind: "zip", upstreamName: origin.Name, upstreamURL: origin.URL, upstreamCommit: origin.Commit,
		upstreamRepository: origin.Repository, upstreamPath: origin.SkillPath}
	if source.upstreamURL == "" {
		source.upstreamURL = origin.Repository
	}
	if target := in.GetTargetSkillId(); target != "" {
		if len(in.GetSkillPaths()) != 1 {
			return nil, errcode.NewPlatformBadRequest()
		}
		skill, err := s.skillhubSvc.GetSkill(ctx, model.SkillID(target))
		if err != nil {
			return nil, catalogError(err)
		}
		_, err = s.saveNewVersion(ctx, skill, filepath.Join(root, in.GetSkillPaths()[0]), source,
			&pb.VersionInput{MakeDefault: in.GetMakeDefault()})
		recordImport(reply, in.GetSkillPaths()[0], true, err)
		return reply, nil
	}
	for _, relative := range in.GetSkillPaths() {
		source.key = sourceKey("zip", in.GetUploadId(), relative)
		added, err := s.saveImport(ctx, filepath.Join(root, relative), source)
		recordImport(reply, relative, added, err)
	}
	return reply, nil
}

func readArchiveOrigin(root string) service.ArchiveOrigin {
	var origin service.ArchiveOrigin
	file, err := os.Open(filepath.Join(root, ".skillhub.json"))
	if err != nil {
		return origin
	}
	defer file.Close()
	if err := json.NewDecoder(io.LimitReader(file, 8192)).Decode(&origin); err != nil {
		return service.ArchiveOrigin{}
	}
	if len(origin.Name) > 120 {
		origin.Name = ""
	}
	if len(origin.URL) > 2048 || !validSourceURL(origin.URL) {
		origin.URL = ""
	}
	if repository, err := service.NormalizeRepository(origin.Repository); err == nil {
		origin.Repository = repository
		if origin.Name == "" {
			origin.Name = strings.Split(strings.TrimPrefix(repository, "https://github.com/"), "/")[0]
		}
	} else {
		origin.Repository = ""
	}
	if len(origin.SkillPath) > 1024 {
		origin.SkillPath = ""
	}
	if !service.IsCommit(origin.Commit) {
		origin.Commit = ""
	}
	return origin
}

func (s skillhubServer) downloadZip(ctx context.Context, in *pb.DownloadZipRequest) (*pb.DownloadZipResponse, error) {
	var data bytes.Buffer
	filename, err := s.writeArchive(ctx, in.GetSkillId(), in.GetVersionId(), &data)
	if err != nil {
		return nil, errcode.WrapPlatformNotFound(err)
	}
	return &pb.DownloadZipResponse{
		Archive:  &httpbody.HttpBody{ContentType: "application/zip", Data: data.Bytes()},
		Filename: filename,
	}, nil
}

func (s skillhubServer) writeArchive(ctx context.Context, skillID, versionID string, target io.Writer) (string, error) {
	s.files.RLock()
	defer s.files.RUnlock()
	version, err := s.ownedVersion(ctx, skillID, versionID)
	if err != nil {
		return "", err
	}
	directory, err := s.versionPath(version)
	if err != nil {
		return "", err
	}
	skill, err := s.skillhubSvc.GetSkill(ctx, model.SkillID(skillID))
	if err != nil {
		return "", err
	}
	origin := service.ArchiveOrigin{Name: skill.UpstreamName(), URL: skill.UpstreamUrl(), Commit: version.UpstreamCommit(),
		Repository: version.UpstreamRepository(), SkillPath: version.UpstreamPath()}
	if version.CommitSha() != "" {
		origin.Repository, origin.SkillPath, origin.Commit = skill.RepositoryUrl(), skill.RepositoryPath(), version.CommitSha()
	}
	return path.Base(version.Name()) + ".zip", service.WriteArchive(target, directory, version.Name(), origin)
}

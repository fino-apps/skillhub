package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	"github.com/google/uuid"
)

const backupLimit = 256 << 20
const backupManifestName = "skillhub-backup.json"

type backupManifest struct {
	Format int `json:"format"`
	repo.LibraryData
}

type backupArchive struct {
	backupManifest
	files map[string]*zip.File
}

func (b backupArchive) snapshot(version model.SkillVersionSnapshot) *zip.File {
	return b.files[backupSnapshotName(version.ID)]
}

func (b backupArchive) hasSnapshot(version model.SkillVersionSnapshot) bool {
	return b.snapshot(version) != nil
}

type BackupImportResult struct {
	Added, Existing int32
}

func backupSnapshotName(id model.SkillVersionID) string {
	return "snapshots/" + string(id) + ".zip"
}

func (s *Library) ExportBackup(ctx context.Context) ([]byte, error) {
	s.writes.Lock()
	defer s.writes.Unlock()

	manifest := backupManifest{Format: 1}
	if err := s.store.Transaction(ctx, func(tx repo.LibraryTx) error {
		var err error
		manifest.LibraryData, err = tx.BackupData()
		return err
	}); err != nil {
		return nil, err
	}

	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	metadata, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	tooLarge := invalid(errors.New("技能库备份超过 256 MiB"))
	var remaining int64 = backupLimit
	writeEntry := func(name string, input io.Reader, size int64) error {
		if size > remaining {
			return tooLarge
		}
		remaining -= size
		entry, err := archive.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		if err != nil {
			return err
		}
		_, err = io.CopyN(entry, input, size)
		return err
	}
	if err := writeEntry(backupManifestName, bytes.NewReader(metadata), int64(len(metadata))); err != nil {
		return nil, err
	}

	for _, version := range manifest.Versions {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		data, err := s.ReadArchive(version)
		if errors.Is(err, repo.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if err := writeEntry(backupSnapshotName(version.ID), bytes.NewReader(data), int64(len(data))); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	if output.Len() > backupLimit {
		return nil, tooLarge
	}
	return output.Bytes(), nil
}

func backupFiles(input io.ReaderAt, size int64) (map[string]*zip.File, error) {
	if size > backupLimit {
		return nil, errors.New("技能库备份超过 256 MiB")
	}
	archive, err := zip.NewReader(input, size)
	if err != nil {
		return nil, err
	}
	files := make(map[string]*zip.File, len(archive.File))
	remaining := uint64(size)
	for _, file := range archive.File {
		if files[file.Name] != nil || file.Method != zip.Store || file.UncompressedSize64 != file.CompressedSize64 ||
			!file.Mode().IsRegular() || file.UncompressedSize64 > remaining {
			return nil, errors.New("技能库备份包含无效或重复的文件")
		}
		remaining -= file.UncompressedSize64
		files[file.Name] = file
	}
	if files[backupManifestName] == nil {
		return nil, errors.New("请选择从 SkillHub 导出的技能库备份")
	}
	return files, nil
}

func readBackupFile(file *zip.File) ([]byte, error) {
	input, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer input.Close()
	return io.ReadAll(input)
}

func parseBackup(data []byte) (backupArchive, error) {
	return parseBackupArchive(bytes.NewReader(data), int64(len(data)))
}

func parseBackupArchive(input io.ReaderAt, size int64) (backupArchive, error) {
	files, err := backupFiles(input, size)
	if err != nil {
		return backupArchive{}, invalid(err)
	}
	metadata, err := readBackupFile(files[backupManifestName])
	if err != nil {
		return backupArchive{}, invalid(err)
	}
	var manifest backupManifest
	if err := json.Unmarshal(metadata, &manifest); err != nil || manifest.Format != 1 {
		return backupArchive{}, invalid(errors.New("无法读取技能库备份，或备份格式版本不受支持"))
	}
	if err := manifest.validate(files); err != nil {
		return backupArchive{}, invalid(err)
	}
	return backupArchive{backupManifest: manifest, files: files}, nil
}

func (s *Library) ImportBackup(ctx context.Context, data []byte) (BackupImportResult, error) {
	backup, err := parseBackup(data)
	if err != nil {
		return BackupImportResult{}, err
	}
	result, err := s.applyBackup(ctx, backup, nil, true)
	return BackupImportResult{Added: result.Added, Existing: result.Skipped}, err
}

func (m backupManifest) validate(files map[string]*zip.File) error {
	skills := make(map[string]bool, len(m.Skills))
	sources := make(map[string]bool)
	for _, skill := range m.Skills {
		id := string(skill.ID)
		if !canonicalBackupID(id) || skills[id] || (skill.SourceKey != "" && sources[skill.SourceKey]) {
			return errors.New("技能库备份包含无效或重复的 Skill")
		}
		skills[id] = true
		sources[skill.SourceKey] = true
	}
	for _, alias := range m.Aliases {
		if alias.SourceKey == "" || sources[alias.SourceKey] || !skills[alias.SkillID] {
			return errors.New("技能库备份包含无效或重复的合并来源")
		}
		sources[alias.SourceKey] = true
	}
	tags := make(map[string]bool, len(m.Tags))
	for _, tag := range m.Tags {
		id := string(tag.ID)
		if !canonicalBackupID(id) || tags[id] || !skills[tag.SkillId] {
			return errors.New("技能库备份包含无效或重复的标签")
		}
		tags[id] = true
	}
	versions := make(map[string]string, len(m.Versions))
	matched := 1
	for _, version := range m.Versions {
		id := string(version.ID)
		if !canonicalBackupID(id) || versions[id] != "" || !skills[version.SkillId] {
			return errors.New("技能库备份包含无效或重复的版本")
		}
		versions[id] = version.SkillId
		if files[backupSnapshotName(version.ID)] != nil {
			matched++
		}
	}
	for _, skill := range m.Skills {
		if skill.DefaultVersionId != "" && versions[skill.DefaultVersionId] != string(skill.ID) {
			return errors.New("技能库备份包含无效的默认版本")
		}
	}
	if matched != len(files) {
		return errors.New("技能库备份包含未关联的快照")
	}
	return nil
}

func canonicalBackupID(id string) bool {
	parsed, err := uuid.Parse(id)
	return err == nil && parsed.String() == id
}

func readBackupSnapshot(file *zip.File, version model.SkillVersionSnapshot) ([]byte, error) {
	data, err := readBackupFile(file)
	if err == nil {
		err = validateBackupSnapshot(data, version.ContentDigest)
	}
	if err != nil {
		return nil, invalid(fmt.Errorf("版本 %s 的快照无效: %w", version.ID, err))
	}
	return data, nil
}

func validateBackupSnapshot(data []byte, expectedDigest string) error {
	return validateSnapshotArchive(bytes.NewReader(data), int64(len(data)), expectedDigest)
}

func validateSnapshotArchive(reader io.ReaderAt, size int64, expectedDigest string) error {
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return err
	}
	if len(archive.File) > 10000 {
		return errors.New("快照文件数量超过限制")
	}
	var remaining uint64 = ArchiveLimit
	seen := make(map[string]bool, len(archive.File))
	digest := sha256.New()
	for _, file := range archive.File {
		if !fs.ValidPath(file.Name) || file.Name == "." || strings.ContainsAny(file.Name, "\\:") ||
			!file.Mode().IsRegular() || seen[file.Name] || (file.Method != zip.Store && file.Method != zip.Deflate) {
			return errors.New("快照包含无效路径或文件")
		}
		seen[file.Name] = true
		if file.UncompressedSize64 > remaining {
			return errors.New("快照解压大小超过限制")
		}

		remaining -= file.UncompressedSize64

		input, err := file.Open()
		if err != nil {
			return err
		}

		fmt.Fprintf(digest, "%q:%d:%d\n", file.Name, file.UncompressedSize64, file.Mode().Perm())
		n, err := io.Copy(digest, io.LimitReader(input, int64(file.UncompressedSize64)+1))
		if err = errors.Join(err, input.Close()); err != nil {
			return err
		}

		if uint64(n) != file.UncompressedSize64 {
			return errors.New("快照文件大小不符")
		}
	}

	if expectedDigest != "" && fmt.Sprintf("%x", digest.Sum(nil)) != expectedDigest {
		return errors.New("快照内容与版本摘要不符")
	}
	return nil
}

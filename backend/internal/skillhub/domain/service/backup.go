package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
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

func backupFiles(data []byte) (map[string]*zip.File, error) {
	if len(data) > backupLimit {
		return nil, errors.New("技能库备份超过 256 MiB")
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	files := make(map[string]*zip.File, len(archive.File))
	remaining := uint64(len(data))
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

func (s *Library) ImportBackup(ctx context.Context, data []byte) (BackupImportResult, error) {
	var result BackupImportResult
	files, err := backupFiles(data)
	if err != nil {
		return result, invalid(err)
	}
	metadata, err := readBackupFile(files[backupManifestName])
	if err != nil {
		return result, invalid(err)
	}
	var manifest backupManifest
	if err := json.Unmarshal(metadata, &manifest); err != nil || manifest.Format != 1 {
		return result, invalid(errors.New("无法读取技能库备份，或备份格式版本不受支持"))
	}
	if err := manifest.validate(files); err != nil {
		return result, invalid(err)
	}

	s.writes.Lock()
	defer s.writes.Unlock()

	var saved []string
	err = s.store.Transaction(ctx, func(tx repo.LibraryTx) error {
		added := make(map[string]bool, len(manifest.Skills))
		for _, skill := range manifest.Skills {
			id := string(skill.ID)
			_, err := tx.Skill(id)
			if errors.Is(err, repo.ErrNotFound) && skill.SourceKey != "" {
				_, err = tx.SkillBySource(skill.SourceKey)
			}
			switch {
			case err == nil:
				result.Existing++
			case errors.Is(err, repo.ErrNotFound):
				if err := tx.CreateSkill(skill); err != nil {
					return err
				}
				added[id] = true
				result.Added++
			default:
				return err
			}
		}
		for _, tag := range manifest.Tags {
			if added[tag.SkillId] {
				if err := tx.CreateTag(tag); err != nil {
					return err
				}
			}
		}
		for _, version := range manifest.Versions {
			if !added[version.SkillId] {
				continue
			}
			if err := tx.CreateVersion(version); err != nil {
				return err
			}
			file := files[backupSnapshotName(version.ID)]
			if file == nil {
				continue
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			data, err := readBackupFile(file)
			if err != nil {
				return invalid(err)
			}
			if err := validateBackupSnapshot(data); err != nil {
				return invalid(fmt.Errorf("版本 %s 的快照无效: %w", version.ID, err))
			}
			id := string(version.ID)
			if err := s.writeSnapshot(id, data); err != nil {
				return err
			}
			saved = append(saved, id)
		}
		return nil
	})
	if err != nil {
		for _, id := range saved {
			_ = os.Remove(s.snapshotPath(id))
		}
		return BackupImportResult{}, err
	}
	return result, nil
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

func validateBackupSnapshot(data []byte) error {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	if len(archive.File) > 10000 {
		return errors.New("快照文件数量超过限制")
	}
	var remaining uint64 = ArchiveLimit
	seen := make(map[string]bool, len(archive.File))
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
	}
	return nil
}

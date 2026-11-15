package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/model"
	"github.com/fino-io/skillhub/backend/internal/skillhub/domain/repo"
	skillpb "github.com/fino-io/skillhub/go/skillhub"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// ImportFile is a file discovered during a read-only import preview.
type ImportFile struct {
	Path      string
	Size      int64
	Digest    string
	Sensitive bool
}

// PreviewLocalSkill returns a stable digest and file list without following links.
func PreviewLocalSkill(path string) ([]ImportFile, string, error) {
	if !filepath.IsAbs(path) {
		return nil, "", fmt.Errorf("skill path must be absolute")
	}
	root, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, "", err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, "", fmt.Errorf("skill path must be a directory")
	}
	if _, ok, err := readSkill(root); err != nil || !ok {
		return nil, "", fmt.Errorf("invalid SKILL.md")
	}
	files := make([]ImportFile, 0)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links are not supported: %s", rel)
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == ".venv" {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported file: %s", rel)
		}
		stat, err := entry.Info()
		if err != nil {
			return err
		}
		if stat.Size() > 20*1024*1024 {
			return fmt.Errorf("file too large: %s", rel)
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		hash := sha256.New()
		_, readErr := io.Copy(hash, file)
		closeErr := file.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		name := strings.ToLower(entry.Name())
		sensitive := name == ".env" || strings.HasPrefix(name, ".env.") || name == "id_rsa" || name == "id_ed25519" || strings.HasSuffix(name, ".pem") || strings.HasSuffix(name, ".key")
		files = append(files, ImportFile{Path: filepath.ToSlash(rel), Size: stat.Size(), Digest: hex.EncodeToString(hash.Sum(nil)), Sensitive: sensitive})
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	hash := sha256.New()
	for _, file := range files {
		fmt.Fprintf(hash, "%s\x00%s\x00", file.Path, file.Digest)
	}
	return files, hex.EncodeToString(hash.Sum(nil)), nil
}

// Intake manages local Skill snapshots in a separate data directory.
type Intake struct {
	root       string
	repository repo.IntakeRepo
	mu         sync.Mutex
}

func NewIntake(root string, repository repo.IntakeRepo) *Intake {
	return &Intake{root: root, repository: repository}
}

// ImportLocal copies exactly the reviewed files and records an initial snapshot.
func (i *Intake) ImportLocal(ctx context.Context, source, expected string, selected []string, version, notes string) (*model.Skill, *model.SkillVersion, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !filepath.IsAbs(source) {
		return nil, nil, fmt.Errorf("skill path must be absolute")
	}
	resolved, err := filepath.EvalSymlinks(source)
	if err != nil {
		return nil, nil, err
	}
	source = resolved
	files, digest, err := PreviewLocalSkill(source)
	if err != nil {
		return nil, nil, err
	}
	if digest != expected {
		return nil, nil, fmt.Errorf("source changed since preview")
	}
	metadata, ok, err := readSkill(source)
	if err != nil || !ok {
		return nil, nil, fmt.Errorf("invalid SKILL.md")
	}
	if len(metadata.Name) > 64 || !validSkillName.MatchString(metadata.Name) {
		return nil, nil, fmt.Errorf("invalid skill name")
	}
	available := make(map[string]ImportFile, len(files))
	for _, file := range files {
		available[file.Path] = file
	}
	chosen := make(map[string]bool, len(selected))
	for _, path := range selected {
		if _, ok := available[path]; !ok || chosen[path] {
			return nil, nil, fmt.Errorf("invalid selected file")
		}
		chosen[path] = true
	}
	if !chosen["SKILL.md"] {
		return nil, nil, fmt.Errorf("SKILL.md must be included")
	}
	if version == "" {
		version = "1.0.0"
	}

	skill, err := model.NewSkill(model.CreateSkillInput{Name: metadata.Name, Description: metadata.Description, Kind: skillpb.SkillKind_SKILL_KIND_PERSONAL})
	if err != nil {
		return nil, nil, err
	}
	initial, err := model.NewSkillVersion(model.CreateSkillVersionInput{SkillId: string(skill.ID()), Version: version, Notes: notes})
	if err != nil {
		return nil, nil, err
	}
	base := filepath.Join("skills", string(skill.ID()))
	currentRel := filepath.Join(base, "current", metadata.Name)
	snapshotRel := filepath.Join(base, "versions", string(initial.ID()), metadata.Name)

	if err := os.MkdirAll(filepath.Join(i.root, ".staging"), 0700); err != nil {
		return nil, nil, err
	}
	stage, err := os.MkdirTemp(filepath.Join(i.root, ".staging"), "import-")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(stage)
	for _, rel := range []string{filepath.Join("current", metadata.Name), filepath.Join("versions", string(initial.ID()), metadata.Name)} {
		for _, file := range files {
			if !chosen[file.Path] {
				continue
			}
			from := filepath.Join(source, filepath.FromSlash(file.Path))
			to := filepath.Join(stage, rel, filepath.FromSlash(file.Path))
			if err := copyReviewedFile(from, to, file.Digest); err != nil {
				return nil, nil, err
			}
		}
	}
	_, after, err := PreviewLocalSkill(source)
	if err != nil || after != digest {
		return nil, nil, fmt.Errorf("source changed since preview")
	}
	_, selectedDigest, err := PreviewLocalSkill(filepath.Join(stage, "current", metadata.Name))
	if err != nil {
		return nil, nil, err
	}
	skillSnapshot := skill.Snapshot()
	skillSnapshot.ContentPath, skillSnapshot.CurrentDigest = currentRel, selectedDigest
	skill = model.RebuildSkill(skillSnapshot)
	versionSnapshot := initial.Snapshot()
	versionSnapshot.SnapshotPath, versionSnapshot.Digest = snapshotRel, selectedDigest
	initial = model.RebuildSkillVersion(versionSnapshot)

	target := filepath.Join(i.root, base)
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return nil, nil, err
	}
	published := false
	err = i.repository.CreateInitial(ctx, skill, initial, func() error {
		if err := os.Rename(stage, target); err != nil {
			return err
		}
		published = true
		return nil
	})
	if err != nil {
		if published {
			_ = os.RemoveAll(target)
		}
		return nil, nil, err
	}
	return skill, initial, nil
}

var validSkillName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func copyReviewedFile(source, target, expected string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("source is not a regular file")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm()&0700)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(out, hash), io.LimitReader(in, 20*1024*1024+1))
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if hex.EncodeToString(hash.Sum(nil)) != expected {
		return fmt.Errorf("source changed since preview")
	}
	return nil
}

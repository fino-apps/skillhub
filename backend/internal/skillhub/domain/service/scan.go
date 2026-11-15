package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ScannedSkill describes one project skill without modifying the project.
type ScannedSkill struct {
	Path        string
	Name        string
	Description string
}

// ScanProject discovers skill directories inside a user-selected project.
func ScanProject(projectPath string) ([]ScannedSkill, error) {
	if !filepath.IsAbs(projectPath) {
		return nil, fmt.Errorf("project path must be absolute")
	}
	root, err := filepath.EvalSymlinks(projectPath)
	if err != nil {
		return nil, fmt.Errorf("resolve project: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project path is not a directory")
	}

	seen := make(map[string]bool)
	results := make([]ScannedSkill, 0)
	for _, name := range []string{".agents/skills", ".codex/skills"} {
		entry := filepath.Join(root, name)
		dir, err := filepath.EvalSymlinks(entry)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("resolve skills directory: %w", err)
		}
		if !within(root, dir) {
			return nil, fmt.Errorf("skills directory is outside project")
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("read skills directory: %w", err)
		}
		for _, item := range entries {
			if strings.HasPrefix(item.Name(), ".") {
				continue
			}
			skillPath, err := filepath.EvalSymlinks(filepath.Join(dir, item.Name()))
			if err != nil {
				return nil, fmt.Errorf("resolve skill: %w", err)
			}
			if !within(root, skillPath) {
				return nil, fmt.Errorf("skill is outside project")
			}
			if seen[skillPath] {
				continue
			}
			info, err := os.Stat(skillPath)
			if err != nil {
				return nil, err
			}
			if !info.IsDir() {
				continue
			}
			skillFile := filepath.Join(skillPath, "SKILL.md")
			realFile, err := filepath.EvalSymlinks(skillFile)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil || !within(skillPath, realFile) {
				return nil, fmt.Errorf("invalid SKILL.md in %s", item.Name())
			}
			data, err := os.ReadFile(realFile)
			if err != nil {
				return nil, err
			}
			if len(data) > 1024*1024 {
				return nil, fmt.Errorf("SKILL.md is too large")
			}
			header := strings.SplitN(string(data), "\n---", 2)
			if len(header) != 2 || !strings.HasPrefix(header[0], "---\n") {
				continue
			}
			var meta struct {
				Name        string `yaml:"name"`
				Description string `yaml:"description"`
			}
			if err := yaml.Unmarshal([]byte(strings.TrimPrefix(header[0], "---\n")), &meta); err != nil {
				continue
			}
			if meta.Name == "" || meta.Description == "" {
				continue
			}
			rel, err := filepath.Rel(root, skillPath)
			if err != nil {
				return nil, err
			}
			seen[skillPath] = true
			results = append(results, ScannedSkill{Path: filepath.ToSlash(rel), Name: meta.Name, Description: meta.Description})
		}
	}
	return results, nil
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

package service

import (
	"fmt"
	"io"
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

// ScanProject discovers skills in a project or reads one directly selected skill directory.
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

	// A direct Skill directory is represented by "." relative to the supplied path.
	if skill, ok, err := readSkill(root); err != nil {
		return nil, err
	} else if ok {
		skill.Path = "."
		return []ScannedSkill{skill}, nil
	}

	seen := make(map[string]bool)
	results := make([]ScannedSkill, 0)
	directories := []string{".agents/skills", ".codex/skills"}
	// A conventional skills directory may also be selected directly.
	if filepath.Base(root) == "skills" {
		directories = append(directories, ".")
	}
	for _, name := range directories {
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
			skill, ok, err := readSkill(skillPath)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			rel, err := filepath.Rel(root, skillPath)
			if err != nil {
				return nil, err
			}
			seen[skillPath] = true
			skill.Path = filepath.ToSlash(rel)
			results = append(results, skill)
		}
	}
	return results, nil
}

// readSkill accepts only a SKILL.md inside the selected directory.
func readSkill(skillPath string) (ScannedSkill, bool, error) {
	skillFile := filepath.Join(skillPath, "SKILL.md")
	realFile, err := filepath.EvalSymlinks(skillFile)
	if os.IsNotExist(err) {
		return ScannedSkill{}, false, nil
	}
	if err != nil || !within(skillPath, realFile) {
		return ScannedSkill{}, false, fmt.Errorf("invalid SKILL.md in %s", filepath.Base(skillPath))
	}
	file, err := os.Open(realFile)
	if err != nil {
		return ScannedSkill{}, false, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 1024*1024+1))
	if err != nil {
		return ScannedSkill{}, false, err
	}
	if len(data) > 1024*1024 {
		return ScannedSkill{}, false, fmt.Errorf("SKILL.md is too large")
	}
	text := strings.TrimPrefix(string(data), "\ufeff")
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return ScannedSkill{}, false, nil
	}
	end := 1
	for end < len(lines) && lines[end] != "---" {
		end++
	}
	if end == len(lines) {
		return ScannedSkill{}, false, nil
	}
	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &meta); err != nil {
		return ScannedSkill{}, false, nil
	}
	if meta.Name == "" || meta.Description == "" {
		return ScannedSkill{}, false, nil
	}
	return ScannedSkill{Name: meta.Name, Description: meta.Description}, true, nil
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

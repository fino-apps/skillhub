package service

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// InstallCommand generates a command to run in the target project.
func InstallCommand(repository, path, commit, name string, global bool) (string, error) {
	repository, err := NormalizeRepository(repository)
	if err != nil {
		return "", err
	}
	path, err = repositoryPath(path)
	if err != nil {
		return "", err
	}
	if !commitPattern.MatchString(commit) || len(name) > 64 || !validSkillName.MatchString(name) {
		return "", fmt.Errorf("invalid version or skill name")
	}
	source := repository + "#" + commit
	if path != "" {
		source = repository + "/tree/" + commit + "/" + path
	}
	return cliInstallCommand(source, name, global), nil
}

func LocalInstallCommand(path, name string, global bool) (string, error) {
	if !filepath.IsAbs(path) || len(name) > 64 || !validSkillName.MatchString(name) {
		return "", fmt.Errorf("invalid local version")
	}
	return cliInstallCommand(path, name, global), nil
}

func cliInstallCommand(source, name string, global bool) string {
	source = strings.ReplaceAll(source, "'", "'\\''")
	command := "npx skills@1.8.1 add '" + source + "' --skill '" + name + "' -a codex --copy"
	if global {
		command += " -g"
	}
	return command
}

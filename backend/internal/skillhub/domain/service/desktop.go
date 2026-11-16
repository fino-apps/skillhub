package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"
)

func SelectDirectory(ctx context.Context) (string, error) {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.CommandContext(ctx, "osascript", "-e", `try
		return POSIX path of (choose folder with prompt "选择文件夹")
	on error number -128
		return ""
	end try`)
	case "linux":
		command = exec.CommandContext(ctx, "zenity", "--file-selection", "--directory", "--title=选择文件夹")
	default:
		return "", errors.New("系统目录选择器不可用，请输入目录路径")
	}
	output, err := command.Output()
	if err != nil {
		var exit *exec.ExitError
		if runtime.GOOS == "linux" && errors.As(err, &exit) && exit.ExitCode() == 1 && len(exit.Stderr) == 0 {
			return "", nil
		}
		return "", errors.New("无法打开系统目录选择器，请输入目录路径")
	}
	return strings.TrimSuffix(string(output), "\n"), nil
}

func SnapshotFiles(root string) ([]string, error) {
	var files []string
	err := fs.WalkDir(os.DirFS(root), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			files = append(files, name)
		}
		return nil
	})
	return files, err
}

type FilePreview struct {
	Text             string
	Size             int64
	Binary, TooLarge bool
}

func PreviewFile(directory, name string) (FilePreview, error) {
	if name == "." || !fs.ValidPath(name) || strings.ContainsAny(name, "\\\x00") {
		return FilePreview{}, errors.New("invalid relative path")
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(directory, filepath.FromSlash(name)))
	if err != nil || resolved != filepath.Join(directory, filepath.FromSlash(name)) {
		return FilePreview{}, errors.New("file is unavailable")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return FilePreview{}, err
	}
	defer root.Close()
	file, err := root.Open(filepath.FromSlash(name))
	if err != nil {
		return FilePreview{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return FilePreview{}, errors.New("file is unavailable")
	}
	result := FilePreview{Size: info.Size(), TooLarge: info.Size() > 1<<20}
	if result.TooLarge {
		return result, nil
	}
	data, err := io.ReadAll(io.LimitReader(file, 1<<20+1))
	if err != nil {
		return FilePreview{}, err
	}
	result.Binary = bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data)
	if !result.Binary {
		result.Text = string(data)
	}
	return result, nil
}

func ExportDirectory(snapshot, destination, name, label string) (string, error) {
	if !filepath.IsAbs(destination) || !validSkillName.MatchString(name) {
		return "", errors.New("invalid export directory or Skill name")
	}
	directory, err := filepath.EvalSymlinks(destination)
	if err != nil {
		return "", err
	}
	if within(snapshot, directory) {
		return "", errors.New("export directory is inside the snapshot")
	}
	target := filepath.Join(directory, name)
	var container string
	if _, err := os.Lstat(target); !errors.Is(err, fs.ErrNotExist) {
		if err != nil {
			return "", err
		}
		container, err = os.MkdirTemp(directory, name+"-"+label+"-")
		if err != nil {
			return "", err
		}
		target = filepath.Join(container, name)
	}

	staging, err := os.MkdirTemp(directory, ".skillhub-export-")
	if err != nil {
		if container != "" {
			_ = os.Remove(container)
		}
		return "", err
	}
	defer os.RemoveAll(staging)
	_, _, err = inspectSkillFiles(snapshot, staging)
	if err == nil {
		if _, check := os.Lstat(target); !errors.Is(check, fs.ErrNotExist) {
			err = fmt.Errorf("export location is no longer available: %s", path.Base(target))
		} else {
			err = os.Rename(staging, target)
		}
	}
	if err != nil && container != "" {
		_ = os.Remove(container)
	}
	return target, err
}

package main

import (
	"cmp"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"

	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/config/source/memory"
)

type libraryPaths struct {
	snapshots, exports, uploads string
}

func configureLibrary() (libraryPaths, error) {
	base := os.Getenv("SKILLHUB_DATA_DIR")
	if base == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return libraryPaths{}, err
		}
		base = filepath.Join(dir, "SkillHub")
	}
	base, err := filepath.Abs(base)
	if err != nil {
		return libraryPaths{}, err
	}
	paths := libraryPaths{
		snapshots: cmp.Or(os.Getenv("SKILLHUB_SNAPSHOT_DIR"), filepath.Join(base, "snapshots")),
		exports:   cmp.Or(os.Getenv("SKILLHUB_EXPORT_DIR"), filepath.Join(base, "exports")),
		uploads:   filepath.Join(base, "uploads"),
	}
	for _, path := range []*string{&paths.snapshots, &paths.exports, &paths.uploads} {
		*path, err = filepath.Abs(*path)
		if err != nil {
			return libraryPaths{}, err
		}
		if err := os.MkdirAll(*path, 0700); err != nil {
			return libraryPaths{}, err
		}
	}
	var configuredDSN string
	if err := config.ScanFrom(&configuredDSN, "db.dsn"); err != nil {
		return libraryPaths{}, err
	}
	override := map[string]any{}
	if filename := os.Getenv("SKILLHUB_DATABASE"); filename != "" || configuredDSN == "" || configuredDSN == "file:skillhub.db?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on" {
		filename, err = filepath.Abs(cmp.Or(filename, filepath.Join(base, "catalog.db")))
		if err != nil {
			return libraryPaths{}, err
		}
		dsn := url.URL{Scheme: "file", Path: filepath.ToSlash(filename), RawQuery: "_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"}
		override["db"] = map[string]any{"dsn": dsn.String()}
	}
	var key string
	if err := config.ScanFrom(&key, "pagination.encodedKey"); err != nil {
		return libraryPaths{}, err
	}
	if key == "" {
		var bytes [32]byte
		if _, err := rand.Read(bytes[:]); err != nil {
			return libraryPaths{}, err
		}
		override["pagination"] = map[string]string{"encodedKey": base64.StdEncoding.EncodeToString(bytes[:])}
	}
	if len(override) == 0 {
		return paths, nil
	}
	data, err := json.Marshal(override)
	if err != nil {
		return libraryPaths{}, err
	}
	return paths, config.Load(memory.NewSource(memory.WithJSON(data)))
}

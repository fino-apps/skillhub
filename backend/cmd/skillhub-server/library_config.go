package main

import (
	"cmp"
	"os"
	"path/filepath"
)

type libraryPaths struct {
	snapshots, exports, uploads string
}

func prepareLibraryPaths() (libraryPaths, error) {
	const base = "skillhub-data"
	paths := libraryPaths{
		snapshots: cmp.Or(os.Getenv("SKILLHUB_SNAPSHOT_DIR"), filepath.Join(base, "snapshots")),
		exports:   cmp.Or(os.Getenv("SKILLHUB_EXPORT_DIR"), filepath.Join(base, "exports")),
		uploads:   filepath.Join(base, "uploads"),
	}

	err := paths.prepare()
	return paths, err
}

func (paths *libraryPaths) prepare() error {
	for _, path := range []*string{&paths.snapshots, &paths.exports, &paths.uploads} {
		absolute, err := filepath.Abs(*path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(absolute, 0700); err != nil {
			return err
		}
		*path = absolute
	}
	return nil
}

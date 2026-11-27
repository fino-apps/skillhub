package service

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestComparisonRejectsOversizedEntry(t *testing.T) {
	file := &zip.File{FileHeader: zip.FileHeader{UncompressedSize64: ArchiveLimit + 1}}
	if _, _, _, err := readForComparison(file); err == nil {
		t.Fatal("expected oversized archive entry to be rejected")
	}
}

func TestBackupRejectsNegativeSize(t *testing.T) {
	if _, err := backupFiles(bytes.NewReader(nil), -1); err == nil {
		t.Fatal("expected negative backup size to be rejected")
	}
}

func TestSnapshotChangesRejectsInvalidID(t *testing.T) {
	var changes snapshotChanges
	if err := changes.write("../escape", nil); err != ErrInvalid {
		t.Fatalf("expected invalid ID error, got %v", err)
	}
}

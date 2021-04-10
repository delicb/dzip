package main

import (
	"bytes"
	"crypto/sha1"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

var dzipExecutable string

func TestDzip(t *testing.T) {
	t.Cleanup(ensureDzip(t))

	tmpDir := t.TempDir()
	beforeZip := filepath.Join(tmpDir, "before.zip")
	afterZip := filepath.Join(tmpDir, "after.zip")

	zipFile(t, beforeZip)
	// change modification time in one of the files
	times := time.Now().Local()
	if err := os.Chtimes("./testdata/file_a", times, times); err != nil {
		t.Fatalf("failed to change modification time of a file: %v", err)
	}
	zipFile(t, afterZip)

	if !compareFileHashes(t, beforeZip, afterZip) {
		t.Fatalf("hashes did not match after chaning modification time of a file")
	}
}

func zipFile(t *testing.T, dest string) {
	if err := exec.Command(dzipExecutable, dest, "./testdata").Run(); err != nil {
		t.Fatalf("failed to zip file: %v", err)
	}
}

func compareFileHashes(t *testing.T, file1, file2 string) bool {
	return bytes.Equal(hashFile(t, file1), hashFile(t, file2))
}

func hashFile(t *testing.T, path string) []byte {
	t.Helper()
	hasher := sha1.New()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file content: %v", err)
	}
	hasher.Write(data)
	return hasher.Sum(nil)
}

func ensureDzip(t *testing.T) func() {
	t.Helper()

	dzipExecutable = "./__test_dzip.test"
	if err := exec.Command("go", "build", "-o", dzipExecutable).Run(); err != nil {
		t.Fatalf("failed to build dzip: %v", err)
	}

	return func() {
		if err := os.Remove(dzipExecutable); err != nil {
			t.Logf("failed to clean up, unable to delete %v: %v", dzipExecutable, err)
		}
	}
}

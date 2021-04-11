package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

var dzipExecutable string

func TestMain(m *testing.M) {
	cleanup, err := ensureDzip()
	if err != nil {
		fmt.Println("### failed creating executable for test ###")
		os.Exit(1)
	}

	testRunResult := m.Run()

	cleanup()

	os.Exit(testRunResult)
}

func TestExpectedContent(t *testing.T) {
	tmpDir := t.TempDir()
	zipFileName := filepath.Join(tmpDir, "out.zip")
	localFileForTest := "testdata/file_a"

	zipFile(t, localFileForTest, zipFileName)

	zipReader, err := zip.OpenReader(zipFileName)
	if err != nil {
		t.Fatalf("failed opening newly created zip file: %v", err)
	}
	defer func(zipReader *zip.ReadCloser) {
		err := zipReader.Close()
		if err != nil {
			t.Fatalf("failed closing zip file as part of cleanup: %v", err)
		}
	}(zipReader)

	f, err := zipReader.Open(localFileForTest)
	if err != nil {
		t.Fatalf("failed opening compressed file in zip: %v", err)
	}

	compressedContent, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("failed reading compressed content: %v", err)
	}

	// read file from test data for comparison
	localContent, err := os.ReadFile(localFileForTest)
	if err != nil {
		t.Fatalf("failed to read content of local file")
	}

	if !bytes.Equal(compressedContent, localContent) {
		t.Fatalf("different content of local and compressed file. Got: %v, expected: %v", string(compressedContent), string(localContent))
	}

}

func TestDeterministicOnModifyTimeChange(t *testing.T) {
	// t.Cleanup(ensureDzip(t))

	tmpDir := t.TempDir()
	beforeZip := filepath.Join(tmpDir, "before.zip")
	afterZip := filepath.Join(tmpDir, "after.zip")

	zipFile(t, "./testdata", beforeZip)
	// change modification time in one of the files
	times := time.Now().Local()
	if err := os.Chtimes("./testdata/file_a", times, times); err != nil {
		t.Fatalf("failed to change modification time of a file: %v", err)
	}
	zipFile(t, "./testdata", afterZip)

	if !compareFileHashes(t, beforeZip, afterZip) {
		t.Fatalf("hashes did not match after chaning modification time of a file")
	}
}

func zipFile(t *testing.T, what, dest string) {
	if out, err := exec.Command(dzipExecutable, dest, what).CombinedOutput(); err != nil {
		t.Fatalf("failed to create zip file: %v\nCombined output is: \n%v", err, string(out))
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

func ensureDzip() (func(), error) {
	// t.Helper()

	dzipExecutable = "./__test_dzip.test"
	if err := exec.Command("go", "build", "-o", dzipExecutable).Run(); err != nil {
		return nil, err
		// t.Fatalf("failed to build dzip: %v", err)
	}

	return func() {
		if err := os.Remove(dzipExecutable); err != nil {

			_, _ = fmt.Fprintf(os.Stderr, "failed to clean up, unable to delete %v: %v", dzipExecutable, err)
		}
	}, nil
}

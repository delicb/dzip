package main

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"dzip": Main,
	}))
}

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"cmpfiles": cmdFiles,
			"chmtime":  cmdChmtime,
			"hasexec":  cmdHasExec,
		},
	})
}

func cmdFiles(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 2 {
		ts.Fatalf("only one argument allowed for sha256")
	}

	hash1 := hashFile(ts, args[0])
	hash2 := hashFile(ts, args[1])
	if neg && hash1 == hash2 {
		ts.Fatalf("found unexpectedly sane hashes")
	}
	if !neg && hash1 != hash2 {
		ts.Fatalf("expected same file hashes, but they are different")
	}
}

func hashFile(ts *testscript.TestScript, path string) string {
	hasher := sha1.New()
	path = ts.MkAbs(path)
	data, err := os.ReadFile(path)
	if err != nil {
		ts.Fatalf("failed to read file: %v", err)
	}
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

func cmdChmtime(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("negation not supported for chmtime")
	}
	var mtime time.Time
	if len(args) == 2 {
		ts.Fatalf("temporary restriction - passing time not supported")
	} else {
		mtime = time.Now().Local().Add(time.Hour)
	}
	ts.Logf("changing mtime")
	if err := os.Chtimes(args[0], mtime, mtime); err != nil {
		ts.Logf("changed mtime to %s", mtime)
		ts.Fatalf("failed to change mtime for file: %v", err)
	}
}

func cmdHasExec(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 1 {
		ts.Fatalf("usage: hasexec <file>")
	}
	stat, err := os.Stat(ts.MkAbs(args[0]))
	if err != nil {
		ts.Fatalf("failed to stat file %v: %v", args[0], err)
	}
	permissions := stat.Mode().Perm()
	if permissions&0111 != 0 {
		if neg {
			ts.Fatalf("file %q has exec unexpected exec flag", args[0])
		}
	} else {
		if !neg {
			ts.Fatalf("file %q does not have exec flag", args[0])
		}
	}

}

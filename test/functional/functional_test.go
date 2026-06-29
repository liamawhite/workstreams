//go:build integration

package functional_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	containerBin     = "/usr/local/bin/workstreams"
	containerWrapper = "/usr/local/share/workstreams.sh"
	wsSentinel       = "__WS_DONE__"
)

// result holds the observable outcome of running a workstreams command via the shell wrapper.
type result struct {
	Stdout      string // stdout from the command (sentinel line stripped)
	Stderr      string // stderr, retained for diagnostic use in t.Fatalf messages
	ExitCode    int
	DirAfter    string // working directory after the command completes; empty if the shell exited
	ShellExited bool   // true if the wrapper called exit (no-arg rm)
}

var globalCtr testcontainers.Container

func TestMain(m *testing.M) {
	ctx := context.Background()

	binPath := filepath.Join(os.TempDir(), "workstreams-linux-functional")
	build := exec.Command("go", "build", "-o", binPath, "../..")
	build.Env = append(os.Environ(), "GOOS=linux", "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n%s\n", err, out)
		os.Exit(1)
	}
	defer os.Remove(binPath)

	var err error
	globalCtr, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:      "alpine:latest",
			Cmd:        []string{"tail", "-f", "/dev/null"},
			WaitingFor: wait.ForExec([]string{"true"}).WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start container: %v\n", err)
		os.Exit(1)
	}
	defer globalCtr.Terminate(ctx)

	if err := globalCtr.CopyFileToContainer(ctx, binPath, containerBin, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "copy binary: %v\n", err)
		os.Exit(1)
	}

	if err := globalCtr.CopyFileToContainer(ctx, "../../workstreams.sh", containerWrapper, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "copy workstreams.sh: %v\n", err)
		os.Exit(1)
	}

	// Hook scripts use bash; Alpine only ships sh by default.
	if _, reader, err := globalCtr.Exec(ctx, []string{"apk", "add", "--no-cache", "bash"}); err != nil {
		fmt.Fprintf(os.Stderr, "install bash: %v\n", err)
		os.Exit(1)
	} else {
		io.Copy(io.Discard, reader)
	}

	os.Exit(m.Run())
}

// testEnv provides an isolated home directory inside the container for one test.
type testEnv struct {
	homeDir string
	tmpDir  string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	base := fmt.Sprintf("/tmp/wstest-%d", time.Now().UnixNano())
	home := base + "/home"
	containerShell(t, fmt.Sprintf("mkdir -p %s", home))
	return &testEnv{homeDir: home, tmpDir: base}
}

// writeFile writes content to a path inside the container.
func (e *testEnv) writeFile(t *testing.T, path, content string) {
	t.Helper()
	containerShell(t, fmt.Sprintf("mkdir -p %s && printf '%%s' %s > %s",
		filepath.Dir(path),
		"'"+strings.ReplaceAll(content, "'", `'\''`)+"'",
		path,
	))
}

// run executes a workstreams command via the shell wrapper inside the container and returns
// the observable outcome: stdout, exit code, working directory after the command, and whether
// the wrapper caused the shell to exit. cwd is the working directory to start from
// (empty defaults to homeDir).
func (e *testEnv) run(t *testing.T, cwd string, args ...string) result {
	t.Helper()

	outFile := e.tmpDir + "/stdout"
	errFile := e.tmpDir + "/stderr"
	dirFile := e.tmpDir + "/dirafter"
	exitFile := e.tmpDir + "/exit"
	scriptFile := e.tmpDir + "/run.sh"

	startDir := e.homeDir
	if cwd != "" {
		startDir = cwd
	}

	var quotedArgs []string
	for _, a := range args {
		quotedArgs = append(quotedArgs, `'`+strings.ReplaceAll(a, `'`, `'\''`)+`'`)
	}

	// The script sources the wrapper so the workstreams shell function handles cd and exit.
	// After the command, we echo a sentinel and capture pwd. If the wrapper called exit,
	// neither line runs, so ShellExited=true and DirAfter is empty.
	// ws_rc captures the workstreams exit code so the script exits with it, not with
	// the exit code of pwd (which would always be 0).
	script := strings.Join([]string{
		"#!/usr/bin/env bash",
		"export HOME=" + e.homeDir,
		"export PATH=/usr/local/bin:$PATH",
		"source " + containerWrapper,
		"cd " + startDir,
		"workstreams " + strings.Join(quotedArgs, " "),
		"ws_rc=$?",
		"echo " + wsSentinel,
		"pwd > " + dirFile,
		"exit $ws_rc",
	}, "\n")

	e.writeFile(t, scriptFile, script)
	containerShell(t, fmt.Sprintf(
		"bash %s >%s 2>%s; echo $? >%s",
		scriptFile, outFile, errFile, exitFile,
	))

	exitStr := strings.TrimSpace(containerShell(t, "cat "+exitFile+" 2>/dev/null || true"))
	exitCode, _ := strconv.Atoi(exitStr)

	rawOut := containerShell(t, "cat "+outFile+" 2>/dev/null || true")
	stderr := strings.TrimSpace(containerShell(t, "cat "+errFile+" 2>/dev/null || true"))
	dirAfter := strings.TrimSpace(containerShell(t, "cat "+dirFile+" 2>/dev/null || true"))

	shellExited := !strings.Contains(rawOut, wsSentinel)
	var kept []string
	for _, line := range strings.Split(rawOut, "\n") {
		if strings.TrimSpace(line) != wsSentinel {
			kept = append(kept, line)
		}
	}

	return result{
		Stdout:      strings.TrimSpace(strings.Join(kept, "\n")),
		Stderr:      stderr,
		ExitCode:    exitCode,
		DirAfter:    dirAfter,
		ShellExited: shellExited,
	}
}

// workstreamDir returns the container path to a named workstream under this env's home.
func (e *testEnv) workstreamDir(name string) string {
	return e.homeDir + "/workstreams/" + name
}

// typeDir returns the container path to a named type under this env's home.
func (e *testEnv) typeDir(name string) string {
	return e.homeDir + "/.workstreams/types/" + name
}

// fileExists reports whether a path exists in the container.
func (e *testEnv) fileExists(t *testing.T, path string) bool {
	t.Helper()
	code, _, _ := globalCtr.Exec(context.Background(), []string{"test", "-e", path})
	return code == 0
}

// readFile returns the content of a file in the container.
func (e *testEnv) readFile(t *testing.T, path string) string {
	t.Helper()
	return containerShell(t, "cat "+path)
}

func containerShell(t *testing.T, script string) string {
	t.Helper()
	_, reader, err := globalCtr.Exec(context.Background(), []string{"sh", "-c", script})
	if err != nil {
		t.Fatalf("container exec %q: %v", script, err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading container output: %v", err)
	}
	return strings.TrimSpace(stripDockerHeader(data))
}

// stripDockerHeader removes the 8-byte Docker stream multiplexing header from each
// chunk so the returned string is the raw command output.
func stripDockerHeader(data []byte) string {
	var out bytes.Buffer
	for len(data) >= 8 {
		size := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
		data = data[8:]
		if size > len(data) {
			size = len(data)
		}
		out.Write(data[:size])
		data = data[size:]
	}
	if out.Len() == 0 {
		return string(data)
	}
	return out.String()
}

// isExecutable reports whether path is executable inside the container.
func isExecutable(t *testing.T, path string) bool {
	t.Helper()
	code, _, _ := globalCtr.Exec(context.Background(), []string{"test", "-x", path})
	return code == 0
}

// --- Tests ---

func TestAdd(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		wantDir     string
		wantErr     bool
	}{
		{"happy path", "My Project", "my-project", false},
		{"single word", "Alpha", "alpha", false},
		{"numbers", "Project 42", "project-42", false},
		{"invalid name", "!!!", "", true},
		{"all spaces", "   ", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			res := env.run(t, "", "add", tt.displayName)

			if tt.wantErr {
				if res.ExitCode == 0 {
					t.Errorf("expected error exit, got 0; stdout=%q", res.Stdout)
				}
				return
			}

			if res.ExitCode != 0 {
				t.Fatalf("unexpected error: stdout=%q stderr=%q", res.Stdout, res.Stderr)
			}

			wantDir := env.workstreamDir(tt.wantDir)
			if !env.fileExists(t, wantDir) {
				t.Errorf("workstream dir %q not created", wantDir)
			}
			if !env.fileExists(t, wantDir+"/config.yaml") {
				t.Errorf("config.yaml not created")
			}

			cfg := env.readFile(t, wantDir+"/config.yaml")
			if !strings.Contains(cfg, "name:") {
				t.Errorf("config.yaml missing name field: %q", cfg)
			}
			if !strings.Contains(cfg, tt.displayName) {
				t.Errorf("config.yaml name does not contain %q: %q", tt.displayName, cfg)
			}

			if res.DirAfter != wantDir {
				t.Errorf("directory after add = %q, want %q", res.DirAfter, wantDir)
			}
		})
	}
}

func TestAddDuplicate(t *testing.T) {
	env := newTestEnv(t)
	if res := env.run(t, "", "add", "My Project"); res.ExitCode != 0 {
		t.Fatalf("first add failed: %s", res.Stderr)
	}
	res := env.run(t, "", "add", "My Project")
	if res.ExitCode == 0 {
		t.Error("expected error adding duplicate workstream, got exit 0")
	}
}

func TestSwitch(t *testing.T) {
	tests := []struct {
		name    string
		create  string // display name to create first (empty = skip)
		switch_ string // dir name to switch to
		wantErr bool
	}{
		{"existing workstream", "My Project", "my-project", false},
		{"nonexistent workstream", "", "ghost", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			if tt.create != "" {
				if res := env.run(t, "", "add", tt.create); res.ExitCode != 0 {
					t.Fatalf("setup add failed: %s", res.Stderr)
				}
			}

			res := env.run(t, "", "switch", tt.switch_)

			if tt.wantErr {
				if res.ExitCode == 0 {
					t.Errorf("expected error, got 0; stdout=%q", res.Stdout)
				}
				return
			}

			if res.ExitCode != 0 {
				t.Fatalf("switch failed: stdout=%q stderr=%q", res.Stdout, res.Stderr)
			}

			wantDir := env.workstreamDir(tt.switch_)
			if res.DirAfter != wantDir {
				t.Errorf("directory after switch = %q, want %q", res.DirAfter, wantDir)
			}
		})
	}
}

func TestTypesAdd(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		wantDir     string
		wantErr     bool
	}{
		{"happy path", "My Type", "my-type", false},
		{"single word", "Golang", "golang", false},
		{"invalid name", "!!!", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			res := env.run(t, "", "types", "add", tt.displayName)

			if tt.wantErr {
				if res.ExitCode == 0 {
					t.Errorf("expected error exit, got 0; stdout=%q", res.Stdout)
				}
				return
			}
			if res.ExitCode != 0 {
				t.Fatalf("unexpected error: stdout=%q stderr=%q", res.Stdout, res.Stderr)
			}

			wantDir := env.typeDir(tt.wantDir)
			if !env.fileExists(t, wantDir) {
				t.Errorf("type dir %q not created", wantDir)
			}
			cfg := env.readFile(t, wantDir+"/config.yaml")
			if !strings.Contains(cfg, tt.displayName) {
				t.Errorf("config.yaml missing display name %q: %q", tt.displayName, cfg)
			}
			for _, hook := range []string{"onInit.sh", "onInitAsync.sh", "onLoad.sh"} {
				path := wantDir + "/" + hook
				if !env.fileExists(t, path) {
					t.Errorf("hook script %q not created", hook)
				}
				if !isExecutable(t, path) {
					t.Errorf("hook script %q is not executable", hook)
				}
			}
		})
	}
}

func TestTypesAddDuplicate(t *testing.T) {
	env := newTestEnv(t)
	if res := env.run(t, "", "types", "add", "My Type"); res.ExitCode != 0 {
		t.Fatalf("first types add failed: %s", res.Stderr)
	}
	res := env.run(t, "", "types", "add", "My Type")
	if res.ExitCode == 0 {
		t.Error("expected error adding duplicate type, got exit 0")
	}
}

func TestTypesRm(t *testing.T) {
	tests := []struct {
		name    string
		create  string
		rm      string
		wantErr bool
	}{
		{"existing type", "My Type", "my-type", false},
		{"nonexistent type", "", "ghost", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			if tt.create != "" {
				if res := env.run(t, "", "types", "add", tt.create); res.ExitCode != 0 {
					t.Fatalf("setup types add failed: %s", res.Stderr)
				}
			}
			res := env.run(t, "", "types", "rm", tt.rm)
			if tt.wantErr {
				if res.ExitCode == 0 {
					t.Errorf("expected error, got 0; stdout=%q", res.Stdout)
				}
				return
			}
			if res.ExitCode != 0 {
				t.Fatalf("types rm failed: stdout=%q stderr=%q", res.Stdout, res.Stderr)
			}
			if env.fileExists(t, env.typeDir(tt.rm)) {
				t.Errorf("type dir still exists after rm")
			}
		})
	}
}

func TestAddWithType(t *testing.T) {
	env := newTestEnv(t)
	if res := env.run(t, "", "types", "add", "My Type"); res.ExitCode != 0 {
		t.Fatalf("setup types add failed: %s", res.Stderr)
	}

	res := env.run(t, "", "add", "My Stream", "--type", "my-type")
	if res.ExitCode != 0 {
		t.Fatalf("add with type failed: stdout=%q stderr=%q", res.Stdout, res.Stderr)
	}

	wantDir := env.workstreamDir("my-stream")
	if !env.fileExists(t, wantDir) {
		t.Errorf("workstream dir %q not created", wantDir)
	}
	cfg := env.readFile(t, wantDir+"/config.yaml")
	if !strings.Contains(cfg, "my-type") {
		t.Errorf("config.yaml missing type field: %q", cfg)
	}
	if res.DirAfter != wantDir {
		t.Errorf("directory after add = %q, want %q", res.DirAfter, wantDir)
	}
}

func TestAddWithNonexistentType(t *testing.T) {
	env := newTestEnv(t)
	res := env.run(t, "", "add", "My Stream", "--type", "ghost")
	if res.ExitCode == 0 {
		t.Error("expected error adding with nonexistent type, got exit 0")
	}
}

func TestSyncInitHook(t *testing.T) {
	env := newTestEnv(t)
	if res := env.run(t, "", "types", "add", "My Type"); res.ExitCode != 0 {
		t.Fatalf("setup types add failed: %s", res.Stderr)
	}
	env.writeFile(t, env.typeDir("my-type")+"/onInit.sh",
		"#!/bin/bash\necho SYNC_MARKER\n")

	res := env.run(t, "", "add", "My Stream", "--type", "my-type")
	if res.ExitCode != 0 {
		t.Fatalf("add failed: stdout=%q stderr=%q", res.Stdout, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "SYNC_MARKER") {
		t.Errorf("SYNC_MARKER not in stdout: %q", res.Stdout)
	}
}

func TestAsyncInitHook(t *testing.T) {
	env := newTestEnv(t)
	if res := env.run(t, "", "types", "add", "My Type"); res.ExitCode != 0 {
		t.Fatalf("setup types add failed: %s", res.Stderr)
	}

	sentinelPath := env.workstreamDir("my-stream") + "/async-sentinel"
	env.writeFile(t, env.typeDir("my-type")+"/onInitAsync.sh",
		"#!/bin/bash\necho ASYNC_DONE > "+sentinelPath+"\n")

	res := env.run(t, "", "add", "My Stream", "--type", "my-type")
	if res.ExitCode != 0 {
		t.Fatalf("add failed: stdout=%q stderr=%q", res.Stdout, res.Stderr)
	}

	// The async script runs detached after the binary exits; poll until it writes the sentinel.
	containerShell(t, fmt.Sprintf(
		"for i in $(seq 20); do test -f %s && break; sleep 1; done", sentinelPath))

	if !env.fileExists(t, sentinelPath) {
		t.Fatalf("async sentinel %q not created within 20s", sentinelPath)
	}
	if content := strings.TrimSpace(env.readFile(t, sentinelPath)); content != "ASYNC_DONE" {
		t.Errorf("async sentinel content = %q, want %q", content, "ASYNC_DONE")
	}
}

func TestLoadHook(t *testing.T) {
	env := newTestEnv(t)
	if res := env.run(t, "", "types", "add", "My Type"); res.ExitCode != 0 {
		t.Fatalf("setup types add failed: %s", res.Stderr)
	}
	if res := env.run(t, "", "add", "My Stream", "--type", "my-type"); res.ExitCode != 0 {
		t.Fatalf("setup add failed: %s", res.Stderr)
	}
	env.writeFile(t, env.typeDir("my-type")+"/onLoad.sh",
		"#!/bin/bash\necho LOAD_MARKER\n")

	res := env.run(t, "", "switch", "my-stream")
	if res.ExitCode != 0 {
		t.Fatalf("switch failed: stdout=%q stderr=%q", res.Stdout, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "LOAD_MARKER") {
		t.Errorf("LOAD_MARKER not in stdout: %q", res.Stdout)
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name       string
		create     string
		remove     string
		fromInside bool
		wantErr    bool
		wantDir    string // expected DirAfter: "base", "home", or ""
	}{
		{"existing workstream", "My Project", "my-project", false, false, "home"},
		{"nonexistent workstream", "", "ghost", false, true, ""},
		{"from inside workstream", "My Project", "my-project", true, false, "base"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			if tt.create != "" {
				if res := env.run(t, "", "add", tt.create); res.ExitCode != 0 {
					t.Fatalf("setup add failed: %s", res.Stderr)
				}
			}

			cwd := ""
			if tt.fromInside {
				cwd = env.workstreamDir(tt.remove)
			}

			res := env.run(t, cwd, "remove", tt.remove)

			if tt.wantErr {
				if res.ExitCode == 0 {
					t.Errorf("expected error, got 0; stdout=%q", res.Stdout)
				}
				return
			}

			if res.ExitCode != 0 {
				t.Fatalf("remove failed: stdout=%q stderr=%q", res.Stdout, res.Stderr)
			}
			if env.fileExists(t, env.workstreamDir(tt.remove)) {
				t.Errorf("workstream dir still exists after remove")
			}

			switch tt.wantDir {
			case "base":
				want := env.homeDir + "/workstreams"
				if res.DirAfter != want {
					t.Errorf("directory after remove = %q, want %q", res.DirAfter, want)
				}
			case "home":
				if res.DirAfter != env.homeDir {
					t.Errorf("directory after remove = %q, want %q (homeDir)", res.DirAfter, env.homeDir)
				}
			}
		})
	}
}

func TestRemoveCurrent(t *testing.T) {
	tests := []struct {
		name    string
		subdir  string // path relative to workstream dir to cd into (empty = root)
		wantErr bool
	}{
		{"from workstream root", "", false},
		{"from subdirectory", "src/pkg", false},
		{"outside workstream", "", true}, // uses homeDir as cwd, not wsDir
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			wsDir := env.workstreamDir("my-project")

			if res := env.run(t, "", "add", "My Project"); res.ExitCode != 0 {
				t.Fatalf("setup add failed: %s", res.Stderr)
			}

			var cwd string
			if tt.wantErr {
				cwd = env.homeDir
			} else {
				cwd = wsDir
				if tt.subdir != "" {
					cwd = wsDir + "/" + tt.subdir
					containerShell(t, "mkdir -p "+cwd)
				}
			}

			res := env.run(t, cwd, "remove")

			if tt.wantErr {
				if res.ExitCode == 0 {
					t.Errorf("expected error, got 0; stdout=%q", res.Stdout)
				}
				return
			}

			if res.ExitCode != 0 {
				t.Fatalf("remove failed: stdout=%q stderr=%q", res.Stdout, res.Stderr)
			}
			if env.fileExists(t, wsDir) {
				t.Errorf("workstream dir still exists after remove")
			}
			if !res.ShellExited {
				t.Errorf("expected shell to exit after removing current workstream, but it continued (DirAfter=%q)", res.DirAfter)
			}
		})
	}
}

func TestCurrent(t *testing.T) {
	tests := []struct {
		name     string
		subdir   string // path relative to workstream dir (empty = root)
		wantName string
		wantErr  bool
	}{
		{"from workstream root", "", "My Project", false},
		{"from subdirectory", "some/nested/dir", "My Project", false},
		{"outside workstream", "", "", true}, // uses homeDir as cwd
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			wsDir := env.workstreamDir("my-project")

			if res := env.run(t, "", "add", "My Project"); res.ExitCode != 0 {
				t.Fatalf("setup add failed: %s", res.Stderr)
			}

			var cwd string
			if tt.wantErr {
				cwd = env.homeDir
			} else {
				cwd = wsDir
				if tt.subdir != "" {
					cwd = wsDir + "/" + tt.subdir
					containerShell(t, "mkdir -p "+cwd)
				}
			}

			res := env.run(t, cwd, "current")

			if tt.wantErr {
				if res.ExitCode == 0 {
					t.Errorf("expected error, got 0; stdout=%q", res.Stdout)
				}
				return
			}

			if res.ExitCode != 0 {
				t.Fatalf("current failed: stdout=%q stderr=%q", res.Stdout, res.Stderr)
			}
			if res.Stdout != tt.wantName {
				t.Errorf("output = %q, want %q", res.Stdout, tt.wantName)
			}
		})
	}
}

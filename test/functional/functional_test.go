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

const containerBin = "/usr/local/bin/workspace"

var globalCtr testcontainers.Container

func TestMain(m *testing.M) {
	ctx := context.Background()

	binPath := filepath.Join(os.TempDir(), "workspace-linux-functional")
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

// templateDir returns the container path to a named template under this env's home.
func (e *testEnv) templateDir(name string) string {
	return e.homeDir + "/workspaces/.templates/" + name
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

// result holds captured output from a single command execution.
type result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// run executes the workspace binary in the container.
// cwd sets the working directory inside the container (empty = default).
func (e *testEnv) run(t *testing.T, cwd string, args ...string) result {
	t.Helper()
	outFile := e.tmpDir + "/stdout"
	errFile := e.tmpDir + "/stderr"

	var quotedArgs []string
	for _, a := range args {
		quotedArgs = append(quotedArgs, "'"+strings.ReplaceAll(a, "'", `'\''`)+"'")
	}

	var script string
	if cwd != "" {
		script = fmt.Sprintf("cd %s && HOME=%s %s %s >%s 2>%s; echo $? >%s/exit",
			cwd, e.homeDir, containerBin, strings.Join(quotedArgs, " "), outFile, errFile, e.tmpDir)
	} else {
		script = fmt.Sprintf("HOME=%s %s %s >%s 2>%s; echo $? >%s/exit",
			e.homeDir, containerBin, strings.Join(quotedArgs, " "), outFile, errFile, e.tmpDir)
	}

	containerShell(t, script)

	exitStr := strings.TrimSpace(containerShell(t, "cat "+e.tmpDir+"/exit"))
	exitCode, _ := strconv.Atoi(exitStr)

	return result{
		Stdout:   strings.TrimSpace(containerShell(t, "cat "+outFile+" 2>/dev/null || true")),
		Stderr:   strings.TrimSpace(containerShell(t, "cat "+errFile+" 2>/dev/null || true")),
		ExitCode: exitCode,
	}
}

// workspaceDir returns the container path to a named workspace under this env's home.
func (e *testEnv) workspaceDir(name string) string {
	return e.homeDir + "/workspaces/" + name
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

// extractChdir returns the path from the first WS_CHDIR:<path> line in s, or empty string.
func extractChdir(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "WS_CHDIR:"); ok {
			return after
		}
	}
	return ""
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
	// Strip Docker multiplexing header bytes if present.
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
					t.Errorf("expected error exit, got 0; stdout=%q stderr=%q", res.Stdout, res.Stderr)
				}
				return
			}

			if res.ExitCode != 0 {
				t.Fatalf("unexpected error: stdout=%q stderr=%q", res.Stdout, res.Stderr)
			}

			wantDir := env.workspaceDir(tt.wantDir)
			if !env.fileExists(t, wantDir) {
				t.Errorf("workspace dir %q not created", wantDir)
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

			if gotChdir := extractChdir(res.Stderr); gotChdir != wantDir {
				t.Errorf("WS_CHDIR path = %q, want %q (full stderr: %q)", gotChdir, wantDir, res.Stderr)
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
		t.Error("expected error adding duplicate workspace, got exit 0")
	}
}

func TestSwitch(t *testing.T) {
	tests := []struct {
		name    string
		create  string // display name to create first (empty = skip)
		switch_ string // dir name to switch to
		wantErr bool
	}{
		{"existing workspace", "My Project", "my-project", false},
		{"nonexistent workspace", "", "ghost", true},
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
					t.Errorf("expected error, got 0; stderr=%q", res.Stderr)
				}
				return
			}

			if res.ExitCode != 0 {
				t.Fatalf("switch failed: stdout=%q stderr=%q", res.Stdout, res.Stderr)
			}

			wantChdir := env.workspaceDir(tt.switch_)
			gotChdir := extractChdir(res.Stderr)
			if gotChdir != wantChdir {
				t.Errorf("WS_CHDIR path = %q, want %q (full stderr: %q)", gotChdir, wantChdir, res.Stderr)
			}
		})
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name       string
		create     string
		remove     string
		fromInside bool
		wantErr    bool
		wantChdir  string // "base", "none", or ""
	}{
		{"existing workspace", "My Project", "my-project", false, false, "none"},
		{"nonexistent workspace", "", "ghost", false, true, ""},
		{"from inside workspace", "My Project", "my-project", true, false, "base"},
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
				cwd = env.workspaceDir(tt.remove)
			}

			res := env.run(t, cwd, "remove", tt.remove)

			if tt.wantErr {
				if res.ExitCode == 0 {
					t.Errorf("expected error, got 0; stderr=%q", res.Stderr)
				}
				return
			}

			if res.ExitCode != 0 {
				t.Fatalf("remove failed: stdout=%q stderr=%q", res.Stdout, res.Stderr)
			}

			if env.fileExists(t, env.workspaceDir(tt.remove)) {
				t.Errorf("workspace dir still exists after remove")
			}

			switch tt.wantChdir {
			case "base":
				wantBase := env.homeDir + "/workspaces"
				if gotChdir := extractChdir(res.Stderr); gotChdir != wantBase {
					t.Errorf("WS_CHDIR path = %q, want %q (full stderr: %q)", gotChdir, wantBase, res.Stderr)
				}
			case "none":
				if got := extractChdir(res.Stderr); got != "" {
					t.Errorf("unexpected WS_CHDIR %q in stderr", got)
				}
			}
		})
	}
}

func TestTemplateAdd(t *testing.T) {
	tests := []struct {
		name     string
		tplName  string
		wantErr  bool
	}{
		{"happy path", "my-tpl", false},
		{"second template", "other-tpl", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			res := env.run(t, "", "template", "add", tt.tplName)
			if tt.wantErr {
				if res.ExitCode == 0 {
					t.Errorf("expected error, got 0; stdout=%q", res.Stdout)
				}
				return
			}
			if res.ExitCode != 0 {
				t.Fatalf("template add failed: %s", res.Stderr)
			}
			if !env.fileExists(t, env.templateDir(tt.tplName)) {
				t.Errorf("template dir %q not created", env.templateDir(tt.tplName))
			}
		})
	}
}

func TestTemplateAddDuplicate(t *testing.T) {
	env := newTestEnv(t)
	if res := env.run(t, "", "template", "add", "my-tpl"); res.ExitCode != 0 {
		t.Fatalf("first template add failed: %s", res.Stderr)
	}
	res := env.run(t, "", "template", "add", "my-tpl")
	if res.ExitCode == 0 {
		t.Error("expected error adding duplicate template, got exit 0")
	}
}

func TestTemplateRm(t *testing.T) {
	env := newTestEnv(t)
	if res := env.run(t, "", "template", "add", "my-tpl"); res.ExitCode != 0 {
		t.Fatalf("setup template add failed: %s", res.Stderr)
	}
	res := env.run(t, "", "template", "rm", "my-tpl")
	if res.ExitCode != 0 {
		t.Fatalf("template rm failed: %s", res.Stderr)
	}
	if env.fileExists(t, env.templateDir("my-tpl")) {
		t.Error("template dir still exists after rm")
	}
}

func TestTemplateRmNonExistent(t *testing.T) {
	env := newTestEnv(t)
	res := env.run(t, "", "template", "rm", "ghost")
	if res.ExitCode == 0 {
		t.Error("expected error removing non-existent template, got exit 0")
	}
}

func TestTemplateList(t *testing.T) {
	env := newTestEnv(t)

	// Empty list should succeed with no output.
	res := env.run(t, "", "template", "list")
	if res.ExitCode != 0 {
		t.Fatalf("template list on empty: %s", res.Stderr)
	}
	if res.Stdout != "" {
		t.Errorf("expected empty output, got %q", res.Stdout)
	}

	for _, name := range []string{"alpha", "beta", "gamma"} {
		if r := env.run(t, "", "template", "add", name); r.ExitCode != 0 {
			t.Fatalf("template add %q: %s", name, r.Stderr)
		}
	}

	res = env.run(t, "", "template", "list")
	if res.ExitCode != 0 {
		t.Fatalf("template list: %s", res.Stderr)
	}
	want := map[string]bool{"alpha": true, "beta": true, "gamma": true}
	for _, line := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !want[line] {
			t.Errorf("unexpected template in list: %q", line)
		}
		delete(want, line)
	}
	for missing := range want {
		t.Errorf("template %q missing from list output", missing)
	}
}

func TestAddWithTemplate(t *testing.T) {
	env := newTestEnv(t)

	// Create a template with a file.
	if res := env.run(t, "", "template", "add", "my-tpl"); res.ExitCode != 0 {
		t.Fatalf("template add: %s", res.Stderr)
	}
	env.writeFile(t, env.templateDir("my-tpl")+"/AGENTS.md", "# Agents\n")

	// Create workspace using the template.
	res := env.run(t, "", "add", "--template", "my-tpl", "My Workspace")
	if res.ExitCode != 0 {
		t.Fatalf("add --template: stdout=%q stderr=%q", res.Stdout, res.Stderr)
	}

	wsDir := env.workspaceDir("my-workspace")
	if !env.fileExists(t, wsDir+"/AGENTS.md") {
		t.Error("template file AGENTS.md not present in workspace")
	}

	cfg := env.readFile(t, wsDir+"/config.yaml")
	if !strings.Contains(cfg, "template: my-tpl") {
		t.Errorf("config.yaml does not record template: %q", cfg)
	}
}

func TestAddWithMissingTemplate(t *testing.T) {
	env := newTestEnv(t)
	res := env.run(t, "", "add", "--template", "nonexistent", "My Workspace")
	if res.ExitCode == 0 {
		t.Error("expected error adding workspace with missing template, got exit 0")
	}
}

func TestTemplateSync(t *testing.T) {
	env := newTestEnv(t)

	// Create template with initial file content.
	if res := env.run(t, "", "template", "add", "my-tpl"); res.ExitCode != 0 {
		t.Fatalf("template add: %s", res.Stderr)
	}
	env.writeFile(t, env.templateDir("my-tpl")+"/AGENTS.md", "v1")

	// Create two workspaces using the template and one without.
	for _, ws := range []string{"Ws One", "Ws Two"} {
		if res := env.run(t, "", "add", "--template", "my-tpl", ws); res.ExitCode != 0 {
			t.Fatalf("add %q: %s", ws, res.Stderr)
		}
	}
	if res := env.run(t, "", "add", "No Template"); res.ExitCode != 0 {
		t.Fatalf("add no-template ws: %s", res.Stderr)
	}
	// Write a workspace-only file to verify it survives sync.
	env.writeFile(t, env.workspaceDir("ws-one")+"/my-own.txt", "mine")

	// Update template and sync.
	env.writeFile(t, env.templateDir("my-tpl")+"/AGENTS.md", "v2")
	res := env.run(t, "", "template", "sync", "my-tpl")
	if res.ExitCode != 0 {
		t.Fatalf("template sync: stdout=%q stderr=%q", res.Stdout, res.Stderr)
	}

	// Template workspaces should have updated file.
	for _, ws := range []string{"ws-one", "ws-two"} {
		content := env.readFile(t, env.workspaceDir(ws)+"/AGENTS.md")
		if strings.TrimSpace(content) != "v2" {
			t.Errorf("workspace %q AGENTS.md = %q, want %q", ws, content, "v2")
		}
	}

	// Workspace-only file must survive.
	if !env.fileExists(t, env.workspaceDir("ws-one")+"/my-own.txt") {
		t.Error("workspace-only file my-own.txt was removed during sync")
	}

	// Non-template workspace must not have AGENTS.md.
	if env.fileExists(t, env.workspaceDir("no-template")+"/AGENTS.md") {
		t.Error("AGENTS.md incorrectly created in non-template workspace")
	}
}

func TestTemplateSyncNonExistent(t *testing.T) {
	env := newTestEnv(t)
	res := env.run(t, "", "template", "sync", "ghost")
	if res.ExitCode == 0 {
		t.Error("expected error syncing non-existent template, got exit 0")
	}
}

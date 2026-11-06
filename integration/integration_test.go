// Package integration exercises the built binary against the real
// search.nixos.org backend.
//
// The tests need network access, so they are skipped under `go test -short`.
// The Nix build runs its checks with -short (see flake.nix), which keeps the
// sandbox offline.
package integration

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binPath is the compiled CLI, built once by TestMain.
var binPath string

func TestMain(m *testing.M) {
	// -short decides whether the binary is worth building, and testing.Short
	// requires the flags to be parsed first.
	flag.Parse()
	if testing.Short() {
		os.Exit(m.Run())
	}

	dir, err := os.MkdirTemp("", "nix-search-integration-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp dir: %v\n", err)
		os.Exit(1)
	}
	binPath = filepath.Join(dir, "nix-search")

	build := exec.Command("go", "build", "-o", binPath,
		"github.com/oscar3170/nix-search/cmd/nix-search")
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build nix-search: %v\n", err)
		os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type result struct {
	stdout string
	stderr string
	code   int
}

// run executes the CLI with extra environment variables.
func run(t *testing.T, env []string, args ...string) result {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test needs network access; run without -short")
	}
	if binPath == "" {
		t.Fatal("nix-search was not built")
	}

	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()

	res := result{stdout: stdout.String(), stderr: stderr.String()}
	if exitErr, ok := err.(*exec.ExitError); ok {
		res.code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("run %v: %v", args, err)
	}
	return res
}

func TestPackageSearch(t *testing.T) {
	res := run(t, nil, "-n", "5", "ripgrep")
	if res.code != 0 {
		t.Fatalf("exit %d, stderr:\n%s", res.code, res.stderr)
	}
	if !strings.Contains(res.stdout, "ripgrep") {
		t.Errorf("output does not mention ripgrep:\n%s", res.stdout)
	}
	if lines := strings.Count(strings.TrimSpace(res.stdout), "\n") + 1; lines > 5 {
		t.Errorf("got %d lines, want at most 5:\n%s", lines, res.stdout)
	}
}

// TestOptionSources checks every documentation set, since each one is selected
// by a different document type in the shared index.
func TestOptionSources(t *testing.T) {
	cases := []struct {
		source string
		query  string
		want   string
	}{
		{"nixos", "nginx enable", "services.nginx.enable"},
		{"home-manager", "programs.git", "programs.git.enable"},
		{"darwin", "system.defaults.dock", "system.defaults.dock.autohide"},
		{"service", "autoconnect", "autoconnect"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			res := run(t, nil, "-n", "5", tc.source, tc.query)
			if res.code != 0 {
				t.Fatalf("exit %d, stderr:\n%s", res.code, res.stderr)
			}
			if !strings.Contains(res.stdout, tc.want) {
				t.Errorf("output does not mention %q:\n%s", tc.want, res.stdout)
			}
		})
	}
}

func TestJSONOutput(t *testing.T) {
	res := run(t, nil, "-o", "json", "-n", "2", "ripgrep")
	if res.code != 0 {
		t.Fatalf("exit %d, stderr:\n%s", res.code, res.stderr)
	}

	// hits holds the bare _source documents, like the Python tool.
	var out struct {
		Total int `json:"total"`
		Hits  []struct {
			AttrName string `json:"package_attr_name"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(res.stdout), &out); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, res.stdout)
	}
	if out.Total < 1 || len(out.Hits) != 2 {
		t.Errorf("total/hits = %d/%d, want at least 1/2", out.Total, len(out.Hits))
	}
	if out.Hits[0].AttrName == "" {
		t.Errorf("first hit has no package_attr_name:\n%s", res.stdout)
	}
}

// TestStaleSchemaVersionFallback pins the alias re-resolution: a bogus mapping
// version must still find the index the backend serves instead of failing.
func TestStaleSchemaVersionFallback(t *testing.T) {
	res := run(t, []string{"NIX_SEARCH_SCHEMA_VERSION=1"}, "-n", "1", "ripgrep")
	if res.code != 0 {
		t.Fatalf("exit %d, stderr:\n%s", res.code, res.stderr)
	}
	if !strings.Contains(res.stdout, "ripgrep") {
		t.Errorf("output does not mention ripgrep:\n%s", res.stdout)
	}
}

func TestUnknownChannel(t *testing.T) {
	res := run(t, nil, "-c", "0.01", "-n", "1", "ripgrep")
	if res.code != 1 {
		t.Fatalf("exit %d, want 1; stdout:\n%s", res.code, res.stdout)
	}
	if !strings.Contains(res.stderr, `unknown channel "0.01"`) {
		t.Errorf("stderr does not report the channel:\n%s", res.stderr)
	}
}

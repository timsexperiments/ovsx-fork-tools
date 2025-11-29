// Package e2e_test provides end-to-end tests for the ovsx-setup tool.
// These tests simulate real-world workflows using local repositories and mocks.
//
// To run these tests, set E2E=true environment variable:
//
//	E2E=true go test -v ./tests/e2e/...
//
// Requirements:
//   - act (GitHub Actions runner)
//   - Docker (for act execution)
//   - git
package e2e_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2E_Sync(t *testing.T) {
	if os.Getenv("E2E") != "true" {
		t.Skip("Skipping E2E test. Set E2E=true to run.")
	}

	forkDir, mocksDir, upstreamDir := setupE2EEnv(t)
	defer os.RemoveAll(forkDir)
	defer os.RemoveAll(mocksDir)
	defer os.RemoveAll(upstreamDir)

	t.Log("Simulating upstream change...")
	runGit(t, upstreamDir, "commit", "--allow-empty", "-m", "Upstream update")

	t.Log("Running sync workflow...")
	// Configure origin to point to the local upstream for git push
	runGit(t, forkDir, "remote", "set-url", "origin", ".e2e/upstream")

	// Inject safe.directory
	syncYamlPath := filepath.Join(forkDir, ".github/workflows/sync.yml")
	injectGitSafeDirectory(t, syncYamlPath)

	runAct(t, forkDir, "sync-pr", "push", "", "sync.yml", mocksDir, upstreamDir)
}

func TestE2E_CI(t *testing.T) {
	if os.Getenv("E2E") != "true" {
		t.Skip("Skipping E2E test. Set E2E=true to run.")
	}

	forkDir, mocksDir, upstreamDir := setupE2EEnv(t)
	defer os.RemoveAll(forkDir)
	defer os.RemoveAll(mocksDir)
	defer os.RemoveAll(upstreamDir)

	t.Log("Running check-version workflow...")

	fixturesDir, _ := filepath.Abs("fixtures")
	prEvent := filepath.Join(fixturesDir, "pr_merged.json") // Using existing PR event

	runAct(t, forkDir, "check-version", "pull_request", prEvent, "check-version.yml", mocksDir, upstreamDir)
}

func TestE2E_Release(t *testing.T) {
	if os.Getenv("E2E") != "true" {
		t.Skip("Skipping E2E test. Set E2E=true to run.")
	}

	forkDir, mocksDir, upstreamDir := setupE2EEnv(t)
	defer os.RemoveAll(forkDir)
	defer os.RemoveAll(mocksDir)
	defer os.RemoveAll(upstreamDir)

	t.Log("Running auto-tag workflow...")

	fixturesDir, _ := filepath.Abs("fixtures")
	prEvent := filepath.Join(fixturesDir, "pr_merged.json")

	// Configure origin to point to the local upstream for git push
	runGit(t, forkDir, "remote", "set-url", "origin", ".e2e/upstream")

	// Inject safe.directory
	autoTagPath := filepath.Join(forkDir, ".github/workflows/auto-tag.yml")
	injectGitSafeDirectory(t, autoTagPath)

	runAct(t, forkDir, "tag-version", "push", prEvent, "auto-tag.yml", mocksDir, upstreamDir)

	t.Log("Running release workflow...")

	// Modify release.yml to use local ovsx mock instead of pnpm dlx
	releaseYamlPath := filepath.Join(forkDir, ".github/workflows/release.yml")
	content, err := os.ReadFile(releaseYamlPath)
	if err != nil {
		t.Fatalf("Failed to read release.yml: %v", err)
	}
	newContent := strings.Replace(string(content), "pnpm dlx ovsx", "ovsx", 1)
	if err := os.WriteFile(releaseYamlPath, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to write release.yml: %v", err)
	}

	injectGitSafeDirectory(t, releaseYamlPath)

	runAct(t, forkDir, "publish", "push", "", "release.yml", mocksDir, upstreamDir)
}

func setupE2EEnv(t *testing.T) (string, string, string) {
	// 1. Setup Directories
	toolPath, _ := filepath.Abs("../../ovsx-setup")

	// Create temp dir inside the project workspace so it's visible to sibling containers (act)
	cwd, _ := os.Getwd()
	tempBase := filepath.Join(cwd, "temp")
	if err := os.MkdirAll(tempBase, 0755); err != nil {
		t.Fatalf("Failed to create temp base dir: %v", err)
	}
	tempDir, err := os.MkdirTemp(tempBase, "ovsx-e2e-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	// Note: We don't defer remove here because we return the dirs.
	// The caller is responsible for cleanup.

	upstreamDir := filepath.Join(tempDir, "upstream")
	forkDir := filepath.Join(tempDir, "fork")
	mocksDir := filepath.Join(tempDir, "mocks")

	mocksSource, _ := filepath.Abs("mocks")

	// Ensure tool is built
	if _, err := os.Stat(toolPath); os.IsNotExist(err) {
		t.Fatalf("ovsx-setup binary not found at %s. Please build it first.", toolPath)
	}

	// 2. Initialize Upstream Repo
	if err := os.MkdirAll(upstreamDir, 0755); err != nil {
		t.Fatalf("Failed to create upstream dir: %v", err)
	}
	runGit(t, upstreamDir, "init", "--initial-branch=main")
	runGit(t, upstreamDir, "config", "user.name", "Upstream User")
	runGit(t, upstreamDir, "config", "user.email", "upstream@example.com")
	runGit(t, upstreamDir, "config", "receive.denyCurrentBranch", "ignore")

	fixturesDir, _ := filepath.Abs("fixtures")
	copyDir(t, filepath.Join(fixturesDir, "extension"), upstreamDir)

	runGit(t, upstreamDir, "add", ".")
	runGit(t, upstreamDir, "commit", "-m", "Initial commit")

	// 3. Initialize Fork Repo
	runGit(t, tempDir, "clone", upstreamDir, "fork")
	runGit(t, forkDir, "config", "user.name", "Fork User")
	runGit(t, forkDir, "config", "user.email", "fork@example.com")

	copyDir(t, mocksSource, mocksDir)

	// 4. Run Setup Tool
	t.Log("Running ovsx-setup...")
	cmd := exec.Command(toolPath, "-p", "test-publisher", "-e", ".")
	cmd.Dir = forkDir
	// Add mocks to PATH for the tool execution (not act)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s:%s", mocksDir, os.Getenv("PATH")))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ovsx-setup failed: %v\nOutput:\n%s", err, out)
	}

	assertFileExists(t, filepath.Join(forkDir, ".github/workflows/sync.yml"))
	assertFileExists(t, filepath.Join(forkDir, ".github/workflows/release.yml"))

	// Commit setup changes
	runGit(t, forkDir, "add", ".")
	runGit(t, forkDir, "commit", "-m", "chore: configure openvsx release workflows")

	return forkDir, mocksDir, upstreamDir
}

func runGit(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed in %s: %v\nOutput:\n%s", args, dir, err, out)
	}
}

func runAct(t *testing.T, dir, job, event, eventPath, workflowFile, mocksDir, upstreamDir string) {
	// act <event> -j <job> -W .github/workflows/<file>

	e2eDir := filepath.Join(dir, ".e2e")
	os.RemoveAll(e2eDir) // Clean up previous run
	if err := os.MkdirAll(e2eDir, 0755); err != nil {
		t.Fatalf("Failed to create .e2e dir: %v", err)
	}

	// Copy mocks
	copyDir(t, mocksDir, filepath.Join(e2eDir, "mocks"))

	copyDir(t, upstreamDir, filepath.Join(e2eDir, "upstream"))

	args := []string{
		"-v",
		event,
		// Use the act-latest image for ubuntu-latest
		"-P", "ubuntu-latest=catthehacker/ubuntu:act-latest",
		// Bind the current working directory to the container
		"--bind",
		"-j", job,
		"-W", filepath.Join(".github/workflows", workflowFile),
		"--container-architecture", "linux/amd64",
		"-b",
		"--env", "GITHUB_TOKEN=mock-token",
		"--var", "PUBLISHER_NAME=test-publisher",
		"--var", "EXTENSION_PATH=.",
		// Inject mocks into PATH.
		// When using -b, act mounts the host directory to the same path in the container.
		// So we must use the absolute path to the mocks.
		"--env", fmt.Sprintf("PATH=%s:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", filepath.Join(e2eDir, "mocks")),
		// Set MOCK_UPSTREAM_URL to the container path (which is also the host path with -b)
		"--env", fmt.Sprintf("MOCK_UPSTREAM_URL=%s", filepath.Join(e2eDir, "upstream")),
	}

	if eventPath != "" {
		args = append(args, "-e", eventPath)
	}

	cmd := exec.Command("act", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("act output:\n%s", output)
		t.Fatalf("act %s failed: %v", job, err)
	}
	t.Logf("act %s success:\n%s", job, output)
}

func copyDir(t *testing.T, src, dst string) {
	t.Logf("Copying from %s to %s", src, dst)
	cmd := exec.Command("cp", "-r", src+"/.", dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to copy fixtures from %s to %s: %v\nOutput: %s", src, dst, err, out)
	}
}

func injectGitSafeDirectory(t *testing.T, path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", path, err)
	}

	// Inject safe.directory config at the start of the job or before git operations
	// We'll try to prepend it to the first 'run' command we find, or just add a setup step.
	// Adding a setup step is safer.

	setupStep := `
      - name: Configure Git Safe Directory
        run: git config --global --add safe.directory '*'
`

	newContent := strings.Replace(string(content), "    steps:", "    steps:"+setupStep, 1)

	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to write %s: %v", path, err)
	}
}

func assertFileExists(t *testing.T, path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("File %s does not exist", path)
	}
}

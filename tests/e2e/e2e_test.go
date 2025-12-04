//go:build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	upstreamRepo = "ovsx-fork-tools/test-extension"
	forkRepo     = "timsexperiments/ovsx-fork-tools-test-extension"
)

func TestRealE2E(t *testing.T) {

	startTime := time.Now()

	// 1. Setup: Clone repos (upstream & fork) to a temp dir
	tempDir, err := os.MkdirTemp("", "real-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	upstreamDir := filepath.Join(tempDir, "upstream")
	forkDir := filepath.Join(tempDir, "fork")

	t.Logf("Setting up E2E in %s", tempDir)
	setupRealE2E(t, upstreamDir, forkDir)

	// 2. Reset:
	//    - Reset fork to the initial commit
	//    - Force push fork
	resetFork(t, forkDir)

	// 3. Install Workflows:
	//    - Run ovsx-setup on fork
	//    - Commit and push workflow files to fork
	runOvsxSetup(t, forkDir)

	// 4. Trigger Upstream Change:
	//    - Bump version in upstream package.json
	//    - Push change to upstream
	newVersion := triggerUpstreamChange(t, upstreamDir)
	t.Logf("New upstream version: %s", newVersion)

	// 5. Sync:
	//    - Trigger sync workflow on fork (workflow_dispatch)
	t.Log("Triggering sync workflow on fork...")
	triggerSyncWorkflow(t)

	// 6. Wait & Merge:
	//    - Wait for PR creation
	t.Log("Waiting for PR creation...")
	var prNumber int
	var prState string
	waitFor(t, "PR creation", func() (bool, error) {
		var found bool
		prNumber, prState, found = findSyncPR(t, forkDir, startTime)
		return found, nil
	})

	//    - Manually merge the sync PR (simulating maintainer action)
	switch prState {
	case "OPEN":
		t.Logf("Merging PR #%d...", prNumber)
		runCommand(t, forkDir, "gh", "pr", "merge", fmt.Sprintf("%d", prNumber), "--merge", "--admin")
	case "MERGED":
		t.Logf("PR #%d was already merged (likely by auto-merge).", prNumber)
		t.Log("Triggering release workflow manually since bot merge doesn't trigger it...")
		runCommand(t, forkDir, "gh", "workflow", "run", "ovsx-fork-tools-release.yml", "--repo", forkRepo, "--ref", "main")
	}

	// 7. Verify Release:
	//    - Wait for release workflow to run and tag to be created
	t.Log("Waiting for release tag...")
	expectedTag := "v" + newVersion
	waitFor(t, fmt.Sprintf("tag %s", expectedTag), func() (bool, error) {
		return checkTag(t, forkDir, expectedTag)
	}, WithTimeout(2*time.Minute))

	t.Logf("Tag %s found!", expectedTag)
}

// --- Helper Functions ---

type waitConfig struct {
	interval time.Duration
	timeout  time.Duration
}

type WaitOption func(*waitConfig)

func WithInterval(d time.Duration) WaitOption {
	return func(c *waitConfig) {
		c.interval = d
	}
}

func WithTimeout(d time.Duration) WaitOption {
	return func(c *waitConfig) {
		c.timeout = d
	}
}

func waitFor(t *testing.T, desc string, check func() (bool, error), opts ...WaitOption) {
	t.Helper()
	cfg := waitConfig{
		interval: 3 * time.Second,
		timeout:  60 * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	deadline := time.Now().Add(cfg.timeout)
	for time.Now().Before(deadline) {
		ok, err := check()
		if err != nil {
			// Don't fail immediately on error, just log and retry (unless it's a fatal error?)
			// For now, treat error as "not ready" but log it.
			t.Logf("Check %q returned error: %v", desc, err)
		} else if ok {
			return
		}
		time.Sleep(cfg.interval)
	}
	t.Fatalf("Timed out waiting for %q after %v", desc, cfg.timeout)
}

func findSyncPR(t *testing.T, forkDir string, minCreatedAt time.Time) (int, string, bool) {
	// Check for all PRs (open and merged)
	output := runCommand(t, forkDir, "gh", "pr", "list", "--state", "all", "--limit", "50", "--json", "number,state,headRefName,createdAt")
	var prs []struct {
		Number    int       `json:"number"`
		State     string    `json:"state"`
		Head      string    `json:"headRefName"`
		CreatedAt time.Time `json:"createdAt"`
	}
	if err := json.Unmarshal([]byte(output), &prs); err != nil {
		t.Logf("Failed to parse PR list: %v", err)
		return 0, "", false
	}

	for _, pr := range prs {
		if pr.Head == "upstream-sync" && pr.CreatedAt.After(minCreatedAt) {
			return pr.Number, pr.State, true
		}
	}
	return 0, "", false
}

func checkTag(t *testing.T, forkDir, expectedTag string) (bool, error) {
	// Use --force to avoid "would clobber existing tag" errors if tag moved or exists locally
	runCommand(t, forkDir, "git", "fetch", "--tags", "--force")
	tagOutput := runCommand(t, forkDir, "git", "tag", "-l", expectedTag)
	return strings.TrimSpace(tagOutput) == expectedTag, nil
}

func setupRealE2E(t *testing.T, upstreamDir, forkDir string) {
	// Clone directly from GitHub to avoid submodule dependency in CI
	upstreamURL := "https://github.com/" + upstreamRepo + ".git"
	forkURL := "https://github.com/" + forkRepo + ".git"

	runCommand(t, ".", "git", "clone", upstreamURL, upstreamDir)
	runCommand(t, ".", "git", "clone", forkURL, forkDir)

	configureGitUser(t, upstreamDir)
	configureGitUser(t, forkDir)

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Fatal("GITHUB_TOKEN environment variable is required")
	}

	upstreamRemote := fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", token, upstreamRepo)
	forkRemote := fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", token, forkRepo)

	runCommand(t, upstreamDir, "git", "remote", "set-url", "origin", upstreamRemote)
	runCommand(t, forkDir, "git", "remote", "set-url", "origin", forkRemote)
}

func configureGitUser(t *testing.T, dir string) {
	runCommand(t, dir, "git", "config", "user.name", "E2E Test")
	runCommand(t, dir, "git", "config", "user.email", "e2e@test.com")
}

func resetFork(t *testing.T, forkDir string) {
	t.Log("Resetting fork...")
	// Delete any existing rulesets that might block force push
	deleteRulesets(t, forkRepo)

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Fatal("GITHUB_TOKEN environment variable is required")
	}
	upstreamRemote := fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", token, upstreamRepo)

	// Add upstream remote to fork
	runCommand(t, forkDir, "git", "remote", "add", "upstream-source", upstreamRemote)

	// Fetch from upstream-source
	runCommand(t, forkDir, "git", "fetch", "upstream-source")

	// Reset to upstream-source/main
	runCommand(t, forkDir, "git", "reset", "--hard", "upstream-source/main")

	// Force push to fork origin
	runCommand(t, forkDir, "git", "push", "--force", "origin", "main")

	// Delete tags on fork to ensure clean state
	deleteRemoteTags(t, forkDir)
}

func deleteRulesets(t *testing.T, repo string) {
	output := runCommand(t, ".", "gh", "api", fmt.Sprintf("repos/%s/rulesets", repo))
	var rulesets []struct {
		ID int `json:"id"`
	}
	json.Unmarshal([]byte(output), &rulesets)

	for _, rs := range rulesets {
		runCommand(t, ".", "gh", "api", fmt.Sprintf("repos/%s/rulesets/%d", repo, rs.ID), "-X", "DELETE")
	}
}

func deleteRemoteTags(t *testing.T, dir string) {
	output := runCommand(t, dir, "git", "ls-remote", "--tags", "origin")
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			ref := parts[1]
			if strings.HasPrefix(ref, "refs/tags/") && !strings.HasSuffix(ref, "^{}") {
				tag := strings.TrimPrefix(ref, "refs/tags/")
				runCommand(t, dir, "git", "push", "--delete", "origin", tag)
			}
		}
	}
}

func runOvsxSetup(t *testing.T, forkDir string) {
	cwd, _ := os.Getwd()
	projectRoot := filepath.Dir(filepath.Dir(cwd))
	if _, err := os.Stat(filepath.Join(projectRoot, "main.go")); os.IsNotExist(err) {
		projectRoot = cwd
	}

	setupBin := filepath.Join(projectRoot, "ovsx-setup-e2e")
	runCommand(t, projectRoot, "go", "build", "-o", setupBin, ".")
	defer os.Remove(setupBin)

	cmd := exec.Command(setupBin, "--extension-path", ".", "--publisher", "TimsExperiments")
	cmd.Dir = forkDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ovsx-setup failed: %v\nOutput: %s", err, output)
	}

	runCommand(t, forkDir, "git", "add", ".")
	status := runCommand(t, forkDir, "git", "status", "--porcelain")
	if status != "" {
		runCommand(t, forkDir, "git", "commit", "-m", "chore: configure openvsx release workflows")
		runCommand(t, forkDir, "git", "push", "origin", "main")
	}
}

func triggerUpstreamChange(t *testing.T, upstreamDir string) string {
	runCommand(t, upstreamDir, "git", "fetch", "origin")
	runCommand(t, upstreamDir, "git", "reset", "--hard", "origin/main")

	packageJsonPath := filepath.Join(upstreamDir, "package.json")
	content, _ := os.ReadFile(packageJsonPath)

	var data map[string]interface{}
	json.Unmarshal(content, &data)

	version := data["version"].(string)
	verParts := strings.Split(version, ".")

	var patch int
	fmt.Sscanf(verParts[2], "%d", &patch)
	newVersion := fmt.Sprintf("%s.%s.%d", verParts[0], verParts[1], patch+1)
	data["version"] = newVersion

	newContent, _ := json.MarshalIndent(data, "", "  ")
	newContent = append(newContent, '\n')
	os.WriteFile(packageJsonPath, newContent, 0644)

	runCommand(t, upstreamDir, "git", "add", "package.json")
	runCommand(t, upstreamDir, "git", "commit", "-m", fmt.Sprintf("chore: bump version to %s", newVersion))
	runCommand(t, upstreamDir, "git", "push", "origin", "main")

	return newVersion
}

func triggerSyncWorkflow(t *testing.T) {
	runCommand(t, ".", "gh", "workflow", "run", "ovsx-fork-tools-sync.yml", "--repo", forkRepo, "--ref", "main")
}

func runCommand(t *testing.T, dir string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		if (name == "gh" && args[0] == "api" && args[2] == "DELETE") ||
			(name == "git" && args[0] == "push" && args[1] == "--delete") {
			return string(output)
		}
		t.Fatalf("Command %s %v failed in %s: %v\nOutput: %s", name, args, dir, err, output)
	}
	return string(output)
}

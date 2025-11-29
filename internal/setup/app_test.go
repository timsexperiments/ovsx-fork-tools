package setup_test

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	app "github.com/timsexperiments/ovsx-fork-tools/internal/setup"
)

type OvsxTest struct {
	name       string
	setup      []Option
	args       []string
	assertions []func(*testing.T, error)
}

type Option func(*testing.T, string)

func NewOvsxSetupTest(name string, opts ...Option) *OvsxTest {
	return &OvsxTest{
		name:  name,
		setup: opts,
	}
}

func (ot *OvsxTest) WithArgs(args ...string) *OvsxTest {
	ot.args = args
	return ot
}

func (ot *OvsxTest) Assert(fn func(*testing.T, error)) *OvsxTest {
	ot.assertions = append(ot.assertions, fn)
	return ot
}

func (ot *OvsxTest) AssertError(contains string) *OvsxTest {
	return ot.Assert(func(t *testing.T, err error) {
		if err == nil {
			t.Error("expected error, got nil")
		} else if !strings.Contains(err.Error(), contains) {
			t.Errorf("expected error containing %q, got %v", contains, err)
		}
	})
}

func (ot *OvsxTest) AssertNoError() *OvsxTest {
	return ot.Assert(func(t *testing.T, err error) {
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func (ot *OvsxTest) AssertWorkflowFilesExist() *OvsxTest {
	return ot.Assert(func(t *testing.T, _ error) {
		workflowDir := filepath.Join(".github", "workflows")
		entries, err := os.ReadDir(workflowDir)
		if err != nil {
			t.Errorf("Failed to read workflow dir: %v", err)
			return
		}
		if len(entries) == 0 {
			t.Error("No files were created in workflow dir")
		}
	})
}

func (ot *OvsxTest) AssertFilesExist() *OvsxTest {
	return ot.Assert(func(t *testing.T, err error) {
		workflowDir := filepath.Join(".github", "workflows")
		expectedFiles := []string{"sync.yml", "release.yml", "auto-tag.yml", "check-version.yml"}
		for _, f := range expectedFiles {
			path := filepath.Join(workflowDir, f)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("File %s was not created", path)
			}
		}
	})
}

func (ot *OvsxTest) AssertFilesNotExist(files ...string) *OvsxTest {
	return ot.Assert(func(t *testing.T, err error) {
		workflowDir := filepath.Join(".github", "workflows")
		for _, f := range files {
			path := filepath.Join(workflowDir, f)
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("File %s should not exist", path)
			}
		}
	})
}

func (ot *OvsxTest) AssertFilesStaged() *OvsxTest {
	return ot.Assert(func(t *testing.T, err error) {
		out, _ := exec.Command("git", "status", "--porcelain").Output()
		if !strings.Contains(string(out), "A  .github/workflows/sync.yml") {
			t.Error("Files were not staged")
		}
	})
}

func (ot *OvsxTest) AssertFilesNotStaged() *OvsxTest {
	return ot.Assert(func(t *testing.T, err error) {
		out, _ := exec.Command("git", "status", "--porcelain").Output()
		if strings.Contains(string(out), "A  .github/workflows/sync.yml") {
			t.Error("Files should not be staged")
		}
	})
}

func (ot *OvsxTest) AssertFileContent(filename, contains string) *OvsxTest {
	return ot.Assert(func(t *testing.T, _ error) {
		workflowDir := filepath.Join(".github", "workflows")
		path := filepath.Join(workflowDir, filename)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", path, err)
			return
		}
		if !strings.Contains(string(content), contains) {
			t.Errorf("File %s does not contain %q", path, contains)
		}
	})
}

func (ot *OvsxTest) Run(t *testing.T) {
	t.Run(ot.name, func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "ovsx-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		os.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(tempDir))
		t.Cleanup(func() { os.Unsetenv("GIT_CEILING_DIRECTORIES") })

		originalWd, _ := os.Getwd()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to chdir: %v", err)
		}
		defer os.Chdir(originalWd)

		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		for _, setup := range ot.setup {
			setup(t, tempDir)
		}
		os.Args = ot.args

		err = app.Run()

		for _, assert := range ot.assertions {
			assert(t, err)
		}
	})
}

func WithEnv(key, value string) Option {
	return func(t *testing.T, dir string) {
		originalValue, wasSet := os.LookupEnv(key)
		os.Setenv(key, value)
		t.Cleanup(func() {
			if wasSet {
				os.Setenv(key, originalValue)
			} else {
				os.Unsetenv(key)
			}
		})
	}
}

func WithGitInit() Option {
	return func(t *testing.T, dir string) {
		if err := exec.Command("git", "init").Run(); err != nil {
			t.Fatalf("Failed to git init: %v", err)
		}
	}
}

func WithDir(path string, perm os.FileMode) Option {
	return func(t *testing.T, dir string) {
		if err := os.MkdirAll(filepath.Join(dir, path), perm); err != nil {
			t.Fatalf("Failed to create dir %s: %v", path, err)
		}
	}
}

func WithDirPermission(path string, perm os.FileMode) Option {
	return func(t *testing.T, dir string) {
		if err := os.Chmod(filepath.Join(dir, path), perm); err != nil {
			t.Fatalf("Failed to chmod %s: %v", path, err)
		}
	}
}

func TestRun(t *testing.T) {
	origPath := os.Getenv("PATH")
	origArgs := os.Args
	t.Cleanup(func() {
		os.Setenv("PATH", origPath)
		os.Args = origArgs
	})

	tests := []*OvsxTest{
		NewOvsxSetupTest("Missing GH CLI", WithEnv("PATH", "")).
			WithArgs("ovsx-setup").
			AssertError("gh not installed"),

		NewOvsxSetupTest("Not a Git Repo", WithEnv("PATH", origPath)).
			WithArgs("ovsx-setup").
			AssertError("not a git repo"),

		NewOvsxSetupTest("Success without Flags", WithEnv("PATH", origPath), WithGitInit()).
			WithArgs("ovsx-setup").
			AssertNoError().
			AssertFilesExist().
			AssertFilesStaged(),

		NewOvsxSetupTest("Success with Flags", WithEnv("PATH", origPath), WithGitInit()).
			WithArgs("ovsx-setup", "-p", "flagpub", "-e", "./flagext").
			AssertNoError().
			AssertFilesExist().
			AssertFilesStaged(),

		NewOvsxSetupTest("Success with Long Flags", WithEnv("PATH", origPath), WithGitInit()).
			WithArgs("ovsx-setup", "--ovsx-publisher", "longpub", "--extension-path", "./longext").
			AssertNoError().
			AssertFilesExist().
			AssertFilesStaged(),

		NewOvsxSetupTest("Write Failure", WithEnv("PATH", origPath), WithGitInit(), WithDir(".github", 0555)).
			WithArgs("ovsx-setup", "-p", "failpub", "-e", "./failext").
			AssertError("permission denied"),

		NewOvsxSetupTest("Write File Failure", WithEnv("PATH", origPath), WithGitInit(), WithDir(".github/workflows", 0755), WithDirPermission(".github/workflows", 0555)).
			WithArgs("ovsx-setup", "-p", "writefail", "-e", "./writefail").
			AssertError("permission denied").
			AssertFilesNotExist().
			AssertFilesNotStaged(),

		NewOvsxSetupTest("Git Add Failure", WithEnv("PATH", origPath), WithGitInit(), WithDirPermission(".git", 0555)).
			WithArgs("ovsx-setup", "-p", "gitfail", "-e", "./gitfail").
			AssertError("failed to git add").
			AssertWorkflowFilesExist().
			AssertFilesNotStaged(),
	}

	for _, test := range tests {
		test.Run(t)
	}
}

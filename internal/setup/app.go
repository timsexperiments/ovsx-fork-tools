package setup

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/timsexperiments/ovsx-fork-tools/internal/setup/workflows"
)

func Run() error {
	fmt.Println("==========================================")
	fmt.Println("   OpenVSX Fork Configuration Assistant   ")
	fmt.Println("==========================================")

	if _, err := exec.LookPath("gh"); err != nil {
		fmt.Println("Error: GitHub CLI (gh) is not installed.")
		fmt.Println("Please install it: https://cli.github.com/")
		return fmt.Errorf("gh not installed")
	}

	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		fmt.Println("Error: This does not look like a git repository.")
		fmt.Println("Please run this command from the root of your forked extension.")
		return fmt.Errorf("not a git repo")
	}

	var publisherFlag string
	var extensionPathFlag string
	flag.StringVar(&publisherFlag, "p", "", "OpenVSX Publisher ID")
	flag.StringVar(&publisherFlag, "publisher", "", "OpenVSX Publisher ID")
	flag.StringVar(&publisherFlag, "ovsx-publisher", "", "OpenVSX Publisher ID")
	flag.StringVar(&extensionPathFlag, "e", "", "Extension Path")
	flag.StringVar(&extensionPathFlag, "extension-path", "", "Extension Path")
	flag.StringVar(&extensionPathFlag, "path", "", "Extension Path")
	flag.StringVar(&extensionPathFlag, "dir", "", "Extension Path")
	flag.Parse()

	publisherName := publisherFlag
	extensionPath := extensionPathFlag

	if publisherName != "" {
		fmt.Printf("Using Publisher ID from flag: %s\n", publisherName)
	}

	if extensionPath != "" {
		fmt.Printf("Using Extension Path from flag: %s\n", extensionPath)
	}

	fmt.Println("\n--- Installing Workflows ---")
	workflowDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		fmt.Printf("Error creating workflow directory: %v\n", err)
		return err
	}

	filesToInstall := map[string][]byte{
		"sync.yml":          workflows.Sync,
		"release.yml":       workflows.Release,
		"auto-tag.yml":      workflows.AutoTag,
		"check-version.yml": workflows.CheckVersion,
	}

	for filename, content := range filesToInstall {
		fileContent := string(content)
		if publisherName != "" {
			fileContent = strings.ReplaceAll(fileContent, `${{ vars.PUBLISHER_NAME }}`, publisherName)
		}
		if extensionPath != "" {
			fileContent = strings.ReplaceAll(fileContent, `${{ vars.EXTENSION_PATH }}`, extensionPath)
		}

		destPath := filepath.Join(workflowDir, filename)
		if err := os.WriteFile(destPath, []byte(fileContent), 0644); err != nil {
			return fmt.Errorf("error writing file %s: %w", destPath, err)
		}
		fmt.Printf("Created %s\n", destPath)

		if err := exec.Command("git", "add", destPath).Run(); err != nil {
			return fmt.Errorf("failed to git add %s: %w", destPath, err)
		}
		fmt.Printf("Staged %s\n", destPath)
	}

	fmt.Println("✅ Workflow files created in .github/workflows/")
	fmt.Println("\n==========================================")
	fmt.Println("   Setup Complete!                        ")
	fmt.Println("==========================================")
	fmt.Println("Next Steps:")
	step := 1
	fmt.Printf("%d. Ensure 'OPEN_VSX_TOKEN' is set in your repository secrets.\n", step)
	step++

	if publisherName == "" {
		fmt.Printf("%d. Set 'PUBLISHER_NAME' in your repository variables (or use -p flag next time).\n", step)
		step++
	}
	if extensionPath == "" {
		fmt.Printf("%d. Set 'EXTENSION_PATH' in your repository variables (or use -e flag next time).\n", step)
		step++
	}

	fmt.Printf("%d. Review the staged changes and commit them:\n", step)
	fmt.Println("   git status")
	fmt.Println("   git commit -m 'chore: configure openvsx release workflows'")
	fmt.Println("")

	return nil
}

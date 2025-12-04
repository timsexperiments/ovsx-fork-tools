# OpenVSX Fork Publisher Kit

This repository contains standard workflows to automate syncing a VS Code extension fork from upstream and publishing it to OpenVSX.

## 🚀 Quick Start (Automated)

If you have the [Go](https://go.dev/dl/) installed, you can setup your fork in seconds.

```bash
go run github.com/timsexperiments/ovsx-fork-tools@latest
```

### Usage

Run the tool from the root of your forked extension repository:

```bash
go run github.com/timsexperiments/ovsx-fork-tools@latest [flags]
```

#### Flags

| Flag                     | Description                                         |
| :----------------------- | :-------------------------------------------------- |
| `-p`, `--publisher`      | Your OpenVSX Publisher ID (e.g. `timsexperiments`)  |
| `-e`, `--extension-path` | Path to the extension within the repo (default `.`) |

**Example:**

```bash
go run github.com/timsexperiments/ovsx-fork-tools@latest -p my-publisher -e ./packages/extension
```

## 🛠 Manual Configuration Guide

If you prefer to set this up manually, you can perform the same steps the tool does using the GitHub CLI (`gh`).

### 1. Install Workflows

Copy the workflow files from this repository to your fork's `.github/workflows/` directory. We recommend using the following names so they are easily identifiable.

1. Copy `internal/setup/workflows/sync.yml` &rarr; `.github/workflows/ovsx-fork-tools-sync.yml`
2. Copy `internal/setup/workflows/release.yml` &rarr; `.github/workflows/ovsx-fork-tools-release.yml`
3. Copy `internal/setup/workflows/check-version.yml` &rarr; `.github/workflows/ovsx-fork-tools-check-version.yml`

### 2. Configure Secrets

Set the `OPEN_VSX_TOKEN` secret. You can get your token from [open-vsx.org/user-settings/tokens](https://open-vsx.org/user-settings/tokens).

```bash
gh secret set OPEN_VSX_TOKEN --body "your_token_here"
```

### 3. Configure Variables

Set the configuration variables required by the workflows.

**Publisher Name:**
The ID of the publisher you created on OpenVSX (e.g., `timsexperiments`).

```bash
gh variable set PUBLISHER_NAME --body "your-publisher-id"
```

**Extension Path:**
The path to the extension package within the repository (usually `.` for the root).

```bash
gh variable set EXTENSION_PATH --body "."
```

### 4. Enable Auto-Merge

The sync workflow relies on auto-merge to seamlessly update your fork.

```bash
gh repo edit --enable-auto-merge
```

## Workflow Details

- **Release to OpenVSX**: Runs on push to `main` or `master` _only_ if the commit message contains "release" or "sync with upstream". It patches the `package.json` with your `PUBLISHER_NAME` on the fly during the build.
- **Sync Upstream**: Runs daily at 3 AM UTC. It automatically detects the parent repository of your fork, pulls changes, and opens a PR.

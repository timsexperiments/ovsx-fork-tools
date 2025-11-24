# OpenVSX Fork Publisher Kit

This repository contains standard workflows to automate syncing a VS Code extension fork from upstream and publishing it to OpenVSX.

## 🚀 Quick Start (Automated)

If you have the [GitHub CLI (`gh`)](https://cli.github.com/) installed, you can setup your fork in seconds.

1. Navigate to your forked repository directory in your terminal.
2. Run the following command (replace `YOUR_USERNAME` with the owner of this kit):

```bash
/bin/bash -c "$(curl -fsSL [https://raw.githubusercontent.com/YOUR_USERNAME/openvsx-publish-kit/main/setup.sh](https://raw.githubusercontent.com/YOUR_USERNAME/openvsx-publish-kit/main/setup.sh))"
```

## 🛠 Manual Configuration Guide

If you prefer to set this up manually, follow the steps below.

### 1. Configure Secrets

Go to **Settings** > **Secrets and variables** > **Actions** > **New repository secret**.

| Name             | Description                                                                                                     |
| :--------------- | :-------------------------------------------------------------------------------------------------------------- |
| `OPEN_VSX_TOKEN` | Your Personal Access Token from [open-vsx.org/user-settings/tokens](https://open-vsx.org/user-settings/tokens). |

### 2. Configure Variables

Go to **Settings** > **Secrets and variables** > **Actions** > **Variables** > **New repository variable**.

| Name             | Value Example                 | Description                                                          |
| :--------------- | :---------------------------- | :------------------------------------------------------------------- |
| `PUBLISHER_NAME` | `timsexperiments`             | The publisher ID you created on OpenVSX.                             |
| `EXTENSION_PATH` | `packages/vscode-tailwindcss` | The path to the extension package within the repo. Use `.` for root. |

### 3. Install Workflows

Copy the files from this repository into your fork:

1. Copy `workflows/sync.yml` to `.github/workflows/sync.yml`
2. Copy `workflows/release.yml` to `.github/workflows/release.yml`

### 4. Enable Auto-Merge

1. Go to **Settings** > **General**.
2. Scroll to **Pull Requests**.
3. Check the box **"Allow auto-merge"**.

## Workflow Details

- **Release to OpenVSX**: Runs on push to `main` or `master` _only_ if the commit message contains "release" or "sync with upstream". It patches the `package.json` with your `PUBLISHER_NAME` on the fly during the build.
- **Sync Upstream**: Runs daily at 3 AM UTC. It automatically detects the parent repository of your fork, pulls changes, and opens a PR.

#!/bin/bash

# OpenVSX Fork Setup Script
# Usage: Run this inside the root of your forked repository.

set -e

echo "=========================================="
echo "   OpenVSX Fork Configuration Assistant   "
echo "=========================================="

# 1. Check for GitHub CLI
if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI (gh) is not installed."
    echo "Please install it: https://cli.github.com/"
    exit 1
fi

# 2. Verify we are in a git repo
if [ ! -d ".git" ]; then
    echo "Error: This does not look like a git repository."
    echo "Please run this command from the root of your forked extension."
    exit 1
fi

# 3. Prompt for Variables
echo ""
echo "--- Configuration ---"
read -p "Enter your OpenVSX Publisher ID (e.g., timsexperiments): " PUBLISHER_NAME
read -p "Enter the path to the extension folder (use '.' for root): " EXTENSION_PATH

if [ -z "$PUBLISHER_NAME" ] || [ -z "$EXTENSION_PATH" ]; then
    echo "Error: Publisher Name and Path are required."
    exit 1
fi

# 4. Set GitHub Variables
echo ""
echo "--- Setting GitHub Repository Variables ---"
gh variable set PUBLISHER_NAME --body "$PUBLISHER_NAME"
gh variable set EXTENSION_PATH --body "$EXTENSION_PATH"
echo "✅ Variables set successfully."

# 5. Check for Secret
echo ""
echo "--- Checking Secrets ---"
if gh secret list | grep -q "OPEN_VSX_TOKEN"; then
    echo "✅ Secret 'OPEN_VSX_TOKEN' already exists."
else
    echo "⚠️  Secret 'OPEN_VSX_TOKEN' not found."
    echo "You must add this manually in Settings > Secrets and variables > Actions."
    echo "Or run: gh secret set OPEN_VSX_TOKEN"
fi

# 6. Create Workflows
echo ""
echo "--- Installing Workflows ---"
mkdir -p .github/workflows

# Base URL for raw files (Update this to point to YOUR repo once published)
# For now, we will write the files directly to ensure they match the guide.
# In a real distribution, you would curl these from your 'openvsx-publish-kit' repo.

# Write sync.yml
cat <<EOF > .github/workflows/sync.yml
name: Sync Upstream

on:
  schedule:
    - cron: '0 3 * * *' # Runs at 3 AM UTC daily
  workflow_dispatch: # Allows manual trigger

jobs:
  sync-pr:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Configure Git
        run: |
          git config --global user.name 'GitHub Action'
          git config --global user.email 'action@github.com'

      - name: Detect Upstream Repository
        id: upstream
        env:
          GH_TOKEN: \${{ secrets.GITHUB_TOKEN }}
        run: |
          # Use GitHub CLI to get the parent repository URL
          PARENT_URL=\$(gh repo view \${{ github.repository }} --json parent --jq '.parent.url')
          
          if [ -z "\$PARENT_URL" ] || [ "\$PARENT_URL" == "null" ]; then
            echo "Error: This repository is not a fork. Cannot sync."
            exit 1
          fi
          
          echo "Detected upstream: \$PARENT_URL"
          
          git remote add upstream \$PARENT_URL
          git fetch upstream

          # Detect upstream default branch (main vs master)
          DEFAULT_BRANCH=\$(git remote show upstream | grep 'HEAD branch' | cut -d' ' -f5)
          echo "Detected upstream default branch: \$DEFAULT_BRANCH"
          
          # Output variables for next steps
          echo "url=\$PARENT_URL" >> \$GITHUB_OUTPUT
          echo "branch=\$DEFAULT_BRANCH" >> \$GITHUB_OUTPUT

      - name: Prepare Merge Branch
        env:
          TARGET_BRANCH: \${{ steps.upstream.outputs.branch }}
        run: |
          git checkout -b upstream-sync
          
          # Merge upstream. 'recursive' handles file additions well.
          git merge upstream/\$TARGET_BRANCH --allow-unrelated-histories -m "chore: sync with upstream"

          # Push to your fork (updates PR if exists)
          git push -f origin upstream-sync

      - name: Create PR & Auto-Merge
        env:
          GH_TOKEN: \${{ secrets.GITHUB_TOKEN }}
          BASE_BRANCH: \${{ steps.upstream.outputs.branch }}
        run: |
          gh pr create \\
            --base \$BASE_BRANCH \\
            --head upstream-sync \\
            --title "chore: sync with upstream" \\
            --body "Automated sync from \${{ steps.upstream.outputs.url }}." \\
            --label "upstream-sync" || echo "PR already exists"

          # Enable auto-merge
          gh pr merge upstream-sync --auto --merge
EOF

# Write release.yml
cat <<EOF > .github/workflows/release.yml
name: Release to OpenVSX

on:
  push:
    branches:
      - main
      - master

jobs:
  publish:
    if: contains(github.event.head_commit.message, 'sync with upstream') || contains(github.event.head_commit.message, 'release')
    runs-on: ubuntu-latest
    env:
      EXTENSION_PATH: \${{ vars.EXTENSION_PATH }}
      PUBLISHER_NAME: \${{ vars.PUBLISHER_NAME }}
      OPEN_VSX_TOKEN: \${{ secrets.OPEN_VSX_TOKEN }}
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - uses: pnpm/action-setup@v3
        with:
          version: ^9.6.0

      - name: Setup Node
        uses: actions/setup-node@v3
        with:
          node-version: 18
          cache: 'pnpm'

      - name: Install Dependencies
        run: pnpm install --frozen-lockfile

      - name: Build Everything
        run: pnpm -r run build

      - name: Patch to TimsExperiments
        run: |
          cd \${{ env.EXTENSION_PATH }}
          
          jq '.publisher = "\${{ env.PUBLISHER_NAME }}"' package.json > package.json.tmp && mv package.json.tmp package.json
          
          echo "Publisher verified as:"
          grep '"publisher":' package.json

      - name: Build & Publish
        env:
          OVSX_PAT: \${{ env.OPEN_VSX_TOKEN }}
        run: |
          cd \${{ env.EXTENSION_PATH }}
          
          pnpm dlx vsce package
          
          pnpm dlx ovsx publish -p \$OVSX_PAT || echo "Publish failed or version already exists."
EOF

echo "✅ Workflow files created in .github/workflows/"
echo ""
echo "=========================================="
echo "   Setup Complete!                        "
echo "=========================================="
echo "Next Steps:"
echo "1. Ensure 'OPEN_VSX_TOKEN' is set in your repository secrets."
echo "2. Commit and push the new workflow files:"
echo "   git add .github/workflows/"
echo "   git commit -m 'chore: configure openvsx release workflows'"
echo "   git push origin main"
echo ""

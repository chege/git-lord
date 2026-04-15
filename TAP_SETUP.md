# Homebrew Tap Setup Guide

This guide walks you through setting up the custom Homebrew tap for automatic releases.

## Overview

Instead of submitting to homebrew-core (which requires approval and is slower), we maintain our own tap at `github.com/chege/homebrew-tap`. GoReleaser automatically updates this tap when new versions are released.

## Prerequisites

- GitHub account with access to create repositories
- Admin access to the `git-lord` repository

## Step 1: Create the Homebrew Tap Repository

1. Go to https://github.com/new
2. Repository name: `homebrew-tap`
3. Description: `Homebrew tap for chege projects`
4. Visibility: Public (required for Homebrew to access it)
5. Initialize with README: Optional
6. Click "Create repository"

## Step 2: Create a Personal Access Token (PAT)

The default `GITHUB_TOKEN` from GitHub Actions can only access the current repository. To push to the tap repository, you need a PAT.

1. Go to https://github.com/settings/tokens/new
2. Token name: `git-lord-tap-token`
3. Expiration: Choose an appropriate duration (recommend 1 year)
4. Scopes: Select `repo` (full control of private repositories)
   - This grants access to read/write repository contents
5. Click "Generate token"
6. **COPY THE TOKEN IMMEDIATELY** - you won't be able to see it again

## Step 3: Add the Token to Repository Secrets

1. Go to https://github.com/chege/git-lord/settings/secrets/actions
2. Click "New repository secret"
3. Name: `TAP_GITHUB_TOKEN`
4. Value: Paste the PAT from Step 2
5. Click "Add secret"

## Step 4: Verify the Setup

### Test the Configuration

```bash
# Install goreleaser locally if not already installed
# brew install goreleaser/tap/goreleaser

# Check the configuration
goreleaser check
```

### Create a Test Release

1. Make a commit following conventional commits format:
   ```bash
   git commit -m "feat: add new feature for testing"
   ```

2. Create and push a tag:
   ```bash
   git tag -a v0.0.0-test -m "Test release"
   git push origin v0.0.0-test
   ```

3. The release workflow will run. Check the Actions tab to see if it succeeds.

4. After successful release, verify the formula was created:
   - Go to https://github.com/chege/homebrew-tap
   - You should see a `Formula/git-lord.rb` file

5. Test installation:
   ```bash
   brew tap chege/tap
   brew install git-lord
   git-lord --version
   ```

6. Clean up the test tag:
   ```bash
   git tag -d v0.0.0-test
   git push --delete origin v0.0.0-test
   # Also delete the release on GitHub
   ```

## How It Works

1. When you push a tag starting with `v`, the release workflow triggers
2. GoReleaser builds binaries for all platforms (Linux, macOS, Windows)
3. GoReleaser creates a GitHub release with the binaries
4. GoReleaser commits a new formula to the `homebrew-tap` repository
5. Users can then `brew install chege/tap/git-lord`

## Troubleshooting

### "Resource not accessible by integration" error

This means the token doesn't have permission to push to the tap repository:
- Verify the PAT has `repo` scope
- Verify `TAP_GITHUB_TOKEN` is set in repository secrets
- Verify the goreleaser.yml uses `token: "{{ .Env.TAP_GITHUB_TOKEN }}"`

### "Repository not found" error

The tap repository doesn't exist or isn't accessible:
- Verify `github.com/chege/homebrew-tap` exists
- Verify it's public (Homebrew requires public taps)

### Formula not updating

Check the release workflow logs:
1. Go to Actions tab in git-lord repo
2. Click the failed release workflow
3. Look for errors in the "Run GoReleaser" step

## Maintenance

### Rotating the PAT

When the PAT expires:
1. Generate a new token (Step 2)
2. Update the `TAP_GITHUB_TOKEN` secret (Step 3)
3. No code changes needed

### Adding Collaborators to the Tap

If others need to push to the tap:
1. Go to https://github.com/chege/homebrew-tap/settings/access
2. Add collaborators with "Write" permission

## Security Notes

- The PAT has write access to repositories. Keep it secure.
- Never commit the PAT to the repository
- Use GitHub Secrets to store the token
- Review the PAT scopes - only `repo` is needed

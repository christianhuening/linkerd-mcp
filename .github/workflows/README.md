# GitHub Actions CI/CD Workflows

This directory contains the CI/CD workflows for the Linkerd MCP server.

## Workflows

### 1. CI Workflow (`ci.yml`)

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop`

**Jobs:**

#### Lint
- Runs `golangci-lint` for code quality checks
- Enforces Go best practices and style guidelines

#### Test
- Runs on Go 1.25 and 1.25 (matrix build)
- Executes all unit tests with race detection
- Generates coverage reports
- Uploads coverage to Codecov
- Creates HTML coverage report artifacts

#### Build
- Builds the binary for verification
- Uploads binary as artifact for inspection
- Verifies build succeeds after tests pass

#### Build Multi-Platform
- Builds binaries for multiple OS/architecture combinations:
  - Linux: amd64, arm64
  - macOS: amd64, arm64
  - Windows: amd64
- Uploads all binaries as artifacts

#### Security Scan
- Runs Trivy vulnerability scanner on codebase
- Runs gosec security scanner for Go code
- Uploads results to GitHub Security tab (SARIF format)

**Artifacts Generated:**
- `coverage-report-go-{version}`: HTML coverage reports
- `linkerd-mcp-binary`: Main build binary
- `linkerd-mcp-{os}-{arch}`: Platform-specific binaries

---

### 2. Docker Workflow (`docker.yml`)

**Triggers:**
- Push to `main` branch
- Tags matching `v*.*.*` (e.g., v1.0.0)
- Pull requests to `main` (build only, no push)

**Jobs:**

#### Build and Push
- Sets up QEMU for multi-architecture builds
- Sets up Docker Buildx
- Builds for linux/amd64 and linux/arm64
- Pushes to GitHub Container Registry (ghcr.io)
- Generates multiple tags:
  - `latest` (for main branch)
  - Version tags (e.g., `1.0.0`, `1.0`, `1`)
  - Branch name tags
  - SHA tags
- Runs Trivy security scan on built image
- Generates SBOM (Software Bill of Materials)

**Image Location:**
```
ghcr.io/<username>/linkerd-mcp:latest
ghcr.io/<username>/linkerd-mcp:v1.0.0
```

**Artifacts Generated:**
- `sbom`: Software Bill of Materials in SPDX JSON format

---

### 3. Release Workflow (`release.yml`)

**Triggers:**
- Tags matching `v*.*.*` (e.g., v1.0.0, v1.2.3)

**Jobs:**

#### Create Release
- Generates changelog from commits
- Creates GitHub Release with:
  - Release notes
  - Installation instructions
  - Docker pull commands
  - Helm installation commands

#### Build Release Binaries
- Builds production binaries for all platforms
- Creates compressed archives (.tar.gz for Unix, .zip for Windows)
- Generates SHA256 checksums for verification
- Uploads to GitHub Release

**Release Assets:**
- `linkerd-mcp-{version}-{os}-{arch}.tar.gz` (or .zip)
- `linkerd-mcp-{version}-{os}-{arch}.tar.gz.sha256`
- `linkerd-mcp-helm-{version}.tgz` (Helm chart)

#### Publish Helm Chart
- Updates Chart.yaml with release version
- Packages Helm chart
- Uploads to GitHub Release

---

## Setting Up Workflows

### Required Secrets

Add these secrets in your GitHub repository settings:

#### Optional (for enhanced features):
- `CODECOV_TOKEN`: For uploading coverage to Codecov
  - Get from: https://codecov.io/
  - Settings → Secrets and variables → Actions → New repository secret

### Required Permissions

The workflows require these permissions (configured in workflow files):
- `contents: write` - For creating releases
- `packages: write` - For pushing to GitHub Container Registry
- `pull-requests: write` - For commenting on PRs
- `security-events: write` - For security scanning results

These are automatically granted via `GITHUB_TOKEN`.

---

## Using the CI/CD System

### Running Tests on Pull Requests

When you create a pull request:
1. CI workflow automatically runs
2. All tests must pass before merge
3. Coverage report is generated
4. Security scans are performed
5. Binaries are built for verification

### Releasing a New Version

```bash
# 1. Ensure main branch is stable
git checkout main
git pull

# 2. Create and push a version tag
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0

# 3. Workflows automatically:
#    - Create GitHub Release
#    - Build binaries for all platforms
#    - Build and push Docker images
#    - Package and upload Helm chart
```

### Accessing Artifacts

#### From Pull Requests/Commits:
1. Go to Actions tab
2. Select the workflow run
3. Scroll to "Artifacts" section
4. Download desired artifacts

#### From Releases:
1. Go to Releases page
2. Find your release version
3. Download from "Assets" section

---

## Workflow Best Practices

### Branch Protection

Configure branch protection for `main`:

```yaml
# Settings → Branches → Add rule

Branch name pattern: main

Required checks:
✓ Require status checks to pass before merging
  - Lint
  - Test (go-1.22)
  - Test (go-1.23)
  - Build
  - Security Scan

✓ Require branches to be up to date before merging
✓ Require pull request reviews before merging (1 reviewer)
✓ Dismiss stale pull request approvals when new commits are pushed
```

### Caching

Workflows use caching to speed up builds:
- Go module cache
- Docker layer cache (via GitHub Actions cache)
- Build cache for repeated builds

### Security

- Security scans run on every commit
- Vulnerability reports uploaded to Security tab
- SBOM generated for Docker images
- All binaries signed with checksums

---

## Troubleshooting

### Tests Failing

```bash
# Run locally first
go test ./internal/... -v

# Check for race conditions
go test ./internal/... -race

# View coverage
go test ./internal/... -cover
```

### Docker Build Failing

```bash
# Test locally
docker build -t linkerd-mcp:test .

# Multi-platform build
docker buildx build --platform linux/amd64,linux/arm64 .
```

### Linting Errors

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run locally
golangci-lint run
```

### Release Failing

Common issues:
- Tag format must be `v*.*.*` (e.g., v1.0.0, not 1.0.0)
- Ensure tag is pushed: `git push origin v1.0.0`
- Check workflow permissions in repository settings

---

## Monitoring

### Status Badges

Add to README.md:

```markdown
[![CI](https://github.com/christianhuening/linkerd-mcp/workflows/CI/badge.svg)](https://github.com/christianhuening/linkerd-mcp/actions/workflows/ci.yml)
[![Docker](https://github.com/christianhuening/linkerd-mcp/workflows/Docker%20Build/badge.svg)](https://github.com/christianhuening/linkerd-mcp/actions/workflows/docker.yml)
[![codecov](https://codecov.io/gh/christianhuening/linkerd-mcp/branch/main/graph/badge.svg)](https://codecov.io/gh/christianhuening/linkerd-mcp)
[![Go Report Card](https://goreportcard.com/badge/github.com/christianhuening/linkerd-mcp)](https://goreportcard.com/report/github.com/christianhuening/linkerd-mcp)
```

### Notifications

Configure in Settings → Notifications:
- Email on workflow failures
- Slack/Discord webhooks for releases

---

## Customization

### Adding New Platforms

Edit `.github/workflows/ci.yml` and `.github/workflows/release.yml`:

```yaml
strategy:
  matrix:
    include:
      # ... existing platforms ...
      - goos: freebsd
        goarch: amd64
```

### Changing Go Version

Update in all workflow files:

```yaml
- name: Set up Go
  uses: actions/setup-go@v6
  with:
    go-version: '1.23'  # Change version here
```

### Adding Custom Build Flags

Edit build commands in workflows:

```yaml
go build -v \
  -ldflags="-s -w -X main.customFlag=value" \
  -tags=custom \
  -o binary .
```

---

## Cost and Usage

### Free Tier Limits (GitHub Actions)

- **Public repositories**: Unlimited minutes
- **Private repositories**: 2,000 minutes/month (Free plan)

### Current Usage

- CI workflow: ~5-10 minutes per run
- Docker workflow: ~10-15 minutes per run
- Release workflow: ~15-20 minutes per run

### Optimization Tips

1. Use caching (already implemented)
2. Run fewer matrix builds on PRs
3. Skip Docker builds on draft PRs
4. Use `paths` filter to run workflows only when relevant files change

Example optimization:

```yaml
on:
  push:
    paths:
      - '**.go'
      - 'go.mod'
      - 'go.sum'
      - '.github/workflows/ci.yml'
```

---

## Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Go in GitHub Actions](https://github.com/actions/setup-go)
- [Docker Build Actions](https://github.com/docker/build-push-action)
- [Security Scanning](https://github.com/aquasecurity/trivy-action)
- [golangci-lint](https://golangci-lint.run/)

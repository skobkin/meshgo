# MeshGo CI/CD Pipeline

This document describes the Continuous Integration and Continuous Deployment pipeline for MeshGo.

## Overview

The project uses **Drone CI** for automated building, testing, and releasing. The pipeline is configured to work with Forgejo (a Gitea fork) for Git hosting and release management.

## Pipeline Structure

### 1. Checks Pipeline

Runs on every push and pull request:

- **Lint**: Uses `golangci-lint` with comprehensive rules
- **Format Check**: Ensures code follows Go formatting standards
- **Build Check**: Verifies the code compiles successfully
- **Test**: Runs all tests with race detection and coverage

### 2. Release Pipeline

Triggered only on Git tags:

- **Multi-platform builds**: Creates binaries for:
  - Linux AMD64
  - Linux ARM64 
  - Windows AMD64
  - macOS Intel (AMD64)
  - macOS Apple Silicon (ARM64)
- **Compression**: All binaries are compressed with ZIP (level 9)
- **Release Creation**: Automatically creates a Forgejo release with:
  - Generated release notes
  - All binary attachments
  - SHA256 and SHA512 checksums
  - Installation instructions

## Configuration Files

### `.drone.yml`
Main CI/CD configuration with two pipelines:
- `checks`: Quality assurance pipeline
- `release`: Automated release pipeline

### `.golangci.yml`
Comprehensive linting configuration with:
- 20+ enabled linters
- Project-specific exclusions
- Performance and security checks

## Setup Requirements

### 1. Secret Configuration
Create the following secrets in your Drone instance:
- `forgejo_token`: API token for Forgejo release creation (with repository write permissions)
- `forgejo_base_url`: Base URL of your Forgejo/Gitea instance (e.g., `https://git.example.com`)

### 2. Automatic Configuration
The pipeline automatically uses these Drone-provided environment variables:
- `DRONE_TAG`: Git tag for release builds
- `DRONE_REPO_LINK`: Repository URL for linking in release notes

The base URL could theoretically be extracted from `DRONE_REPO_LINK`, but using a secret is more reliable and explicit.

### 3. Repository Settings
Ensure your repository has:
- Drone CI enabled
- Webhook configured to trigger builds
- Proper access permissions for the build user

## Local Development

### Prerequisites
```bash
# Install development tools
make install-tools

# Or manually:
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Common Commands
```bash
# Run all checks locally (same as CI)
make ci-check

# Individual checks
make fmt-check    # Format verification
make lint         # Run linter
make test         # Run tests
make build        # Build binary

# Development cycle
make dev          # Format, vet, test, build, and run
```

### Build for All Platforms
```bash
# Local cross-compilation
make build-all

# Package for distribution
make package
```

## Release Process

### 1. Prepare Release
```bash
# Ensure all changes are committed
git status

# Run local checks
make ci-check

# Update version in relevant files if needed
```

### 2. Create Release
```bash
# Create and push tag (this triggers the release pipeline)
make tag VERSION=v1.2.3

# Or manually:
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3
```

### 3. Monitor Pipeline
- Check Drone dashboard for build status
- Verify all platforms build successfully
- Confirm release is created in Forgejo

## Pipeline Features

### Security
- No secrets exposed in logs
- Minimal permissions required
- Isolated build environments

### Performance
- Parallel execution where possible
- Efficient caching of dependencies
- Optimized binary sizes with `-ldflags "-s -w"`

### Reliability
- Comprehensive testing on every change
- Multi-platform compatibility verification
- Automated checksum generation

## Troubleshooting

### Common Issues

1. **CGO Build Failures**
   - macOS builds use `CGO_ENABLED=0` to avoid cross-compilation issues
   - Linux ARM64 requires cross-compiler setup
   - Windows requires MinGW toolchain

2. **Lint Failures**
   - Run `make lint` locally to see issues
   - Check `.golangci.yml` for configuration
   - Some GUI-specific code may need exclusions

3. **Test Failures**
   - Ensure all dependencies are available
   - GUI tests may require X11 forwarding in containers
   - Race conditions should be fixed, not ignored

4. **Release Failures**
   - Verify both `forgejo_token` and `forgejo_base_url` secrets are configured
   - Check that the API token has proper repository write permissions
   - Ensure tag follows semantic versioning
   - Verify the base URL is correct and accessible from Drone

### Debug Commands
```bash
# Test pipeline steps locally
docker run --rm -v $(pwd):/workspace -w /workspace golangci/golangci-lint:v1.60.3 golangci-lint run

# Check build dependencies
go list -m all
go mod why -m <module-name>

# Verify cross-compilation
GOOS=windows GOARCH=amd64 go build ./cmd/meshgo
```

## Pipeline Maintenance

### Updating Dependencies
```bash
# Update Go version in .drone.yml
# Update golangci-lint version in .drone.yml
# Update linter rules in .golangci.yml
```

### Adding New Platforms
1. Add build step to `.drone.yml`
2. Add platform to release file list
3. Test cross-compilation locally
4. Update documentation

### Performance Optimization
- Monitor build times in Drone dashboard
- Optimize Docker image choices
- Consider build caching strategies
- Profile resource usage

## Best Practices

1. **Always test locally** before pushing
2. **Use semantic versioning** for releases
3. **Keep CI configuration simple** and maintainable
4. **Monitor build performance** and optimize as needed
5. **Document any special requirements** for new features
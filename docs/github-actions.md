# GitHub Actions CI/CD Pipeline

This document describes the GitHub Actions workflow for automated testing, building, and publishing of the `k8s-external-ip-powerdns` Docker images.

## Workflow Overview

The CI/CD pipeline (`.github/workflows/docker-publish.yml`) automates the entire process from code push to Docker image publication in GitHub Container Registry (GHCR).

### Workflow Jobs

1. **Test Job** (`test`)
   - Runs on `ubuntu-latest`
   - Sets up Go 1.22
   - Downloads dependencies with caching
   - Runs all Go tests
   - Builds the binary to verify compilation

2. **Build and Push Job** (`build-and-push`)
   - Depends on successful test completion
   - Builds multi-architecture Docker images
   - Publishes to GitHub Container Registry
   - Generates proper metadata and tags

3. **Security Scan Job** (`security-scan`)
   - Runs Trivy vulnerability scanner
   - Uploads results to GitHub Security tab
   - Provides security insights

## Triggers

The workflow is triggered by:

### Push Events
- **Main branch**: Creates `latest` tag
- **Develop branch**: Creates `develop` tag
- **Version tags**: Creates semantic version tags

### Pull Requests
- **To main branch**: Creates `pr-<number>` tags for testing

## Container Registry

Images are published to GitHub Container Registry (GHCR):
- **Registry**: `ghcr.io`
- **Image path**: `ghcr.io/<username>/<repository>/k8s-external-ip-powerdns`

### Authentication
- Uses `GITHUB_TOKEN` (automatically provided)
- No manual secrets configuration required
- Permissions handled via workflow permissions

## Tagging Strategy

| Event Type | Tag Pattern | Example |
|------------|-------------|---------|
| Main branch push | `latest` | `latest` |
| Develop branch push | `develop` | `develop` |
| Version tag | `v<version>`, `v<major>.<minor>`, `v<major>` | `v1.2.3`, `v1.2`, `v1` |
| Pull request | `pr-<number>` | `pr-42` |
| Feature branch | `<branch-name>` | `feature-new-auth` |

## Multi-Architecture Support

The workflow builds images for multiple architectures:
- `linux/amd64` (Intel/AMD 64-bit)
- `linux/arm64` (ARM 64-bit, including Apple Silicon)

This ensures compatibility with various deployment environments including:
- Traditional x86_64 servers
- ARM-based cloud instances (AWS Graviton, etc.)
- Apple Silicon development machines
- Raspberry Pi and other ARM devices

## Build Arguments

The workflow passes build-time information to the Docker build:

```dockerfile
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
```

These are automatically populated with:
- `VERSION`: Git tag or branch name
- `COMMIT`: Short Git commit hash
- `BUILD_DATE`: ISO 8601 timestamp

## Caching Strategy

The workflow uses multiple caching layers for optimal performance:

### Go Module Cache
- Caches `~/go/pkg/mod`
- Key: `${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}`
- Significantly speeds up dependency downloads

### Docker Build Cache
- Uses GitHub Actions cache (`type=gha`)
- Caches Docker build layers between runs
- Reduces build time for subsequent runs

## Security Features

### Trivy Vulnerability Scanning
- Scans the final Docker image for vulnerabilities
- Uploads results to GitHub Security tab
- Runs after successful image build
- Provides SARIF format output for detailed analysis

### Container Security
- Runs as non-root user (UID 1000)
- Uses minimal Alpine Linux base image
- Includes only necessary certificates and timezone data
- No unnecessary packages or tools in final image

## Usage Examples

### Publishing a New Version

1. **Create and push a version tag:**
   ```bash
   git tag v1.2.3
   git push origin v1.2.3
   ```

2. **The workflow will automatically:**
   - Run tests
   - Build multi-arch images
   - Tag with `v1.2.3`, `v1.2`, `v1`, and `latest`
   - Publish to GHCR
   - Run security scan

### Using Published Images

```bash
# Pull the latest version
docker pull ghcr.io/<username>/<repository>/k8s-external-ip-powerdns:latest

# Pull a specific version
docker pull ghcr.io/<username>/<repository>/k8s-external-ip-powerdns:v1.2.3

# Pull develop branch build
docker pull ghcr.io/<username>/<repository>/k8s-external-ip-powerdns:develop
```

### In Kubernetes Deployments

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8s-external-ip-powerdns
spec:
  template:
    spec:
      containers:
      - name: k8s-external-ip-powerdns
        image: ghcr.io/<username>/<repository>/k8s-external-ip-powerdns:v1.2.3
        # ... rest of container spec
```

## Troubleshooting

### Build Failures

1. **Test failures**: Check the test job logs for specific test failures
2. **Build failures**: Verify Dockerfile syntax and build context
3. **Permission issues**: Ensure repository has appropriate permissions for GHCR

### Image Pull Issues

1. **Authentication**: Ensure you're authenticated with GHCR:
   ```bash
   echo $GITHUB_TOKEN | docker login ghcr.io -u <username> --password-stdin
   ```

2. **Image not found**: Verify the exact image path and tag
3. **Architecture mismatch**: Ensure you're pulling the correct architecture

### Security Scan Failures

1. **High vulnerabilities**: Review Trivy scan results in GitHub Security tab
2. **False positives**: Consider using Trivy ignore files if needed
3. **Base image updates**: Update Alpine version in Dockerfile for security patches

## Workflow Permissions

The workflow requires these permissions:
- `contents: read` - Read repository contents
- `packages: write` - Publish to GitHub Packages/GHCR
- `id-token: write` - OIDC token for enhanced security
- `security-events: write` - Upload security scan results

These are automatically configured in the workflow file.

## Monitoring and Notifications

### GitHub Actions UI
- View workflow runs in the "Actions" tab
- Monitor build status and logs
- Download artifacts if needed

### Security Tab
- View Trivy scan results
- Monitor vulnerability trends
- Set up security advisories if needed

### Branch Protection
Consider setting up branch protection rules:
- Require status checks (workflow success)
- Require up-to-date branches
- Restrict force pushes

## Best Practices

1. **Version Management**
   - Use semantic versioning for tags
   - Create releases with detailed changelogs
   - Test changes in feature branches first

2. **Security**
   - Regularly update dependencies
   - Monitor security scan results
   - Keep base images updated

3. **Performance**
   - Leverage caching effectively
   - Minimize Docker image layers
   - Use multi-stage builds efficiently

4. **Monitoring**
   - Set up notifications for failed builds
   - Monitor image pull metrics
   - Review security scan results regularly

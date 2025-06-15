# Kubernetes External IP PowerDNS Sync

A Go application that automatically synchronizes external IP addresses from Kubernetes nodes (via the `k3s.io/external-ip` annotation) with PowerDNS records using the official [go-powerdns](https://github.com/joeig/go-powerdns) library. It supports both IPv4 and IPv6 addresses, creating A and AAAA records respectively.

## Features

- ✅ Fetches external IP addresses from Kubernetes node annotations
- ✅ Supports both IPv4 and IPv6 addresses
- ✅ Creates A records for IPv4 addresses
- ✅ Creates AAAA records for IPv6 addresses
- ✅ Handles single and multiple IP addresses per node
- ✅ Deduplicates IP addresses across nodes
- ✅ Configurable sync interval and DNS TTL
- ✅ PowerDNS API integration using the official go-powerdns library
- ✅ Kubernetes RBAC support
- ✅ Docker containerization
- ✅ Health monitoring and logging
- ✅ Automatic FQDN handling

## Prerequisites

- Kubernetes cluster with nodes having the `k3s.io/external-ip` annotation
- PowerDNS server with API enabled
- PowerDNS API key
- Go 1.21+ (for building from source)

## Configuration

The application is configured via environment variables:

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `POWERDNS_URL` | Yes | PowerDNS API base URL | `http://powerdns-api:8081` |
| `POWERDNS_API_KEY` | Yes | PowerDNS API key | `your-secret-api-key` |
| `POWERDNS_VHOST` | No | PowerDNS virtual host (default: localhost) | `localhost` |
| `DNS_ZONE` | Yes | DNS zone to update | `example.com.` |
| `DNS_RECORD` | Yes | DNS record name to update | `cluster.example.com.` |
| `DNS_TTL` | No | DNS record TTL (default: 300s) | `300s`, `5m` |
| `SYNC_INTERVAL` | No | Sync interval (default: 30s) | `60s`, `5m`, `1h` |
| `KUBECONFIG` | No | Path to kubeconfig file | `/path/to/kubeconfig` |

## Node Annotation Format

The application looks for the `k3s.io/external-ip` annotation on Kubernetes nodes. The annotation value should contain comma-separated IP addresses:

```yaml
apiVersion: v1
kind: Node
metadata:
  name: node1
  annotations:
    k3s.io/external-ip: "152.67.73.95,2603:c022:5:1e00:a452:9f75:7f83:3a88"
```

### Supported Formats

- Single IPv4: `152.67.73.95`
- Single IPv6: `2603:c022:5:1e00:a452:9f75:7f83:3a88`
- Multiple IPs: `152.67.73.95,2603:c022:5:1e00:a452:9f75:7f83:3a88`
- Mixed with spaces: `152.67.73.95, 2603:c022:5:1e00:a452:9f75:7f83:3a88`

## Building

### From Source

```bash
# Clone the repository
git clone <repository-url>
cd k8s-external-ip-powerdns

# Download dependencies
go mod download

# Build the binary
go build -o k8s-external-ip-powerdns .
```

### Docker Image

#### Using Pre-built Images

Pre-built multi-architecture Docker images are automatically published to GitHub Container Registry via GitHub Actions:

```bash
# Pull the latest image
docker pull ghcr.io/[username]/k8s-external-ip-powerdns:latest

# Or pull a specific version
docker pull ghcr.io/[username]/k8s-external-ip-powerdns:v1.0.0
```

#### Building Locally

```bash
# Build the Docker image
docker build -t k8s-external-ip-powerdns:latest .

# Or use the Makefile
make docker-build
```

#### GitHub Actions Workflow

The project includes a comprehensive GitHub Actions workflow (`.github/workflows/docker-publish.yml`) that:

- **Tests**: Runs all Go tests before building
- **Multi-arch builds**: Builds for both `linux/amd64` and `linux/arm64`
- **Auto-publishing**: Publishes to GitHub Container Registry (`ghcr.io`)
- **Tagging strategy**:
  - `latest` for main branch
  - `v1.2.3` for semantic version tags
  - `main`, `develop` for branch builds
  - `pr-123` for pull requests
- **Security scanning**: Runs Trivy vulnerability scans
- **Build optimization**: Uses GitHub Actions cache

**Triggering builds:**
- Push to `main` or `develop` branches
- Create version tags (e.g., `git tag v1.0.0 && git push origin v1.0.0`)
- Open pull requests to `main`

## Running

### Local Development

```bash
# Set environment variables
export POWERDNS_URL="http://your-powerdns-server:8081"
export POWERDNS_API_KEY="your-api-key"
export POWERDNS_VHOST="localhost"
export DNS_ZONE="example.com."
export DNS_RECORD="cluster.example.com."
export DNS_TTL="300s"
export SYNC_INTERVAL="30s"

# Run the application
./k8s-external-ip-powerdns
```

### Kubernetes Deployment

1. Update the configuration in `k8s-deployment.yaml`:
   ```yaml
   # Update ConfigMap with your values
   data:
     POWERDNS_URL: "http://your-powerdns-server:8081"
     POWERDNS_VHOST: "localhost"
     DNS_ZONE: "your-domain.com."
     DNS_RECORD: "cluster.your-domain.com."
     DNS_TTL: "300s"
   
   # Update Secret with your API key (base64 encoded)
   data:
     POWERDNS_API_KEY: <base64-encoded-api-key>
   ```

2. Apply the manifests:
   ```bash
   kubectl apply -f k8s-deployment.yaml
   ```

3. Check the logs:
   ```bash
   kubectl logs -f deployment/k8s-external-ip-powerdns
   ```

## How It Works

1. **Startup Validation**: The application performs startup checks to verify:
   - Kubernetes API connectivity and proper RBAC permissions
   - PowerDNS API connectivity and zone access
   - Configuration validity
   
   **Note**: The application will fail fast with clear error messages if any startup checks fail, preventing misconfigured deployments from running indefinitely.

2. **Node Discovery**: The application connects to the Kubernetes API and lists all nodes in the cluster.

3. **IP Extraction**: For each node, it checks for the `k3s.io/external-ip` annotation and parses the comma-separated IP addresses.

4. **IP Classification**: Each IP address is classified as IPv4 or IPv6 using Go's `net.ParseIP()` function.

5. **Deduplication**: IP addresses are deduplicated across all nodes to avoid creating duplicate DNS records.

6. **DNS Record Creation**: 
   - IPv4 addresses are used to create/update A records
   - IPv6 addresses are used to create/update AAAA records
   - If no IPs of a particular type are found, existing records of that type are deleted

7. **DNS Record Updates**: 
   - IPv4 addresses are used to create/update A records via `pdns.Records.Change()`
   - IPv6 addresses are used to create/update AAAA records via `pdns.Records.Change()`
   - If no IPs of a particular type are found, existing records are deleted via `pdns.Records.Delete()`

8. **PowerDNS API**: The application uses the official `go-powerdns` library to interact with PowerDNS's REST API, providing robust error handling and type safety.

9. **Periodic Sync**: The process repeats at the configured interval to ensure DNS records stay in sync with the cluster state.

## PowerDNS API Integration

The application uses the official [`go-powerdns`](https://github.com/joeig/go-powerdns) library for PowerDNS integration. This provides:

- **Type Safety**: Strongly typed API calls with proper error handling
- **Authentication**: Automatic X-API-Key header management
- **Record Management**: High-level methods for DNS record operations
- **Context Support**: Full Go context support for cancellation and timeouts
- **Robust Error Handling**: Detailed error messages and HTTP status code handling

### Key Operations

- **A Records**: Uses `pdns.Records.Change()` for IPv4 addresses
- **AAAA Records**: Uses `pdns.Records.Change()` for IPv6 addresses  
- **Record Deletion**: Uses `pdns.Records.Delete()` when no IPs of a type exist
- **Zone Validation**: Verifies zone existence at startup
- **Connection Testing**: Tests API connectivity before starting sync loop

### Example API Usage

The application uses the go-powerdns library like this:

```go
// Initialize client
pdns := powerdns.New(
    "http://powerdns-server:8081",
    "localhost", 
    powerdns.WithAPIKey("your-api-key"),
)

// Update A record
err := pdns.Records.Change(ctx, "example.com.", "cluster.example.com.", 
    powerdns.RRTypeA, 300, []string{"192.168.1.1"})

// Update AAAA record  
err := pdns.Records.Change(ctx, "example.com.", "cluster.example.com.",
    powerdns.RRTypeAAAA, 300, []string{"2001:db8::1"})

// Delete record
err := pdns.Records.Delete(ctx, "example.com.", "cluster.example.com.", 
    powerdns.RRTypeA)
```

## RBAC Permissions

The application requires the following Kubernetes permissions:

```yaml
rules:
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["get", "list", "watch"]
```

These permissions allow the application to:
- List all nodes in the cluster
- Read node metadata and annotations
- Watch for changes to nodes (for future enhancements)

## Logging

The application provides detailed logging for monitoring and debugging:

```
2025/06/15 10:30:00 Starting k8s-external-ip-powerdns sync service...
2025/06/15 10:30:00 Configuration loaded:
2025/06/15 10:30:00   PowerDNS URL: http://powerdns-api:8081
2025/06/15 10:30:00   PowerDNS VHost: localhost
2025/06/15 10:30:00   DNS Zone: example.com.
2025/06/15 10:30:00   DNS Record: cluster.example.com.
2025/06/15 10:30:00   DNS TTL: 300 seconds
2025/06/15 10:30:00   Sync Interval: 30s
2025/06/15 10:30:00 Connected to PowerDNS API, found 1 servers
2025/06/15 10:30:00 Successfully verified DNS zone: example.com.
2025/06/15 10:30:00 Fetching external IP addresses from Kubernetes nodes...
2025/06/15 10:30:00 Found external IPs for node node1: 152.67.73.95,2603:c022:5:1e00:a452:9f75:7f83:3a88
2025/06/15 10:30:00 Found 2 external IP addresses:
2025/06/15 10:30:00   152.67.73.95 (IPv4)
2025/06/15 10:30:00   2603:c022:5:1e00:a452:9f75:7f83:3a88 (IPv6)
2025/06/15 10:30:00 Updating DNS records for cluster.example.com. in zone example.com....
2025/06/15 10:30:00 Updating A record for cluster.example.com. with 1 IPv4 addresses
2025/06/15 10:30:00 Successfully updated A record for cluster.example.com.
2025/06/15 10:30:00 Updating AAAA record for cluster.example.com. with 1 IPv6 addresses
2025/06/15 10:30:00 Successfully updated AAAA record for cluster.example.com.
2025/06/15 10:30:00 Starting periodic sync every 30s...
```

## Error Handling

The application handles various error scenarios:

- **Invalid IP addresses**: Logs warnings and skips invalid IPs
- **Missing annotations**: Logs info messages for nodes without external IP annotations
- **PowerDNS API errors**: Logs detailed error messages including HTTP status codes
- **Kubernetes API errors**: Graceful handling of connection issues and retries
- **Configuration errors**: Fails fast with clear error messages

## Security Considerations

- **API Key Security**: PowerDNS API key is stored in Kubernetes secrets
- **RBAC**: Minimal required permissions for Kubernetes access
- **Container Security**: Runs as non-root user with read-only filesystem
- **Network Security**: Only requires outbound HTTPS to PowerDNS API

## Troubleshooting

### Common Issues

1. **Service Account Permissions Error**:
   ```
   Failed to access Kubernetes nodes - check service account permissions: nodes is forbidden: 
   User "system:serviceaccount:default:default" cannot list resource "nodes" in API group "" at the cluster scope
   ```
   
   **Solution**: The application requires proper RBAC permissions to access Kubernetes nodes. Apply the complete RBAC configuration:
   ```bash
   kubectl apply -f k8s-deployment.yaml
   ```
   
   The required permissions are:
   ```yaml
   rules:
   - apiGroups: [""]
     resources: ["nodes"]
     verbs: ["get", "list", "watch"]
   ```

2. **No external IPs found**:
   - Check if nodes have the `k3s.io/external-ip` annotation
   - Verify the annotation value format

3. **PowerDNS API errors**:
   - Verify PowerDNS URL and API key
   - Check PowerDNS API is enabled and accessible
   - Ensure the DNS zone exists in PowerDNS

4. **DNS records not updating**:
   - Check PowerDNS logs for API requests
   - Verify the zone and record names are correct
   - Ensure PowerDNS has proper backend configuration

### Debug Commands

```bash
# Check node annotations
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.k3s\.io/external-ip}{"\n"}{end}'

# Check application logs
kubectl logs -f deployment/k8s-external-ip-powerdns

# Test PowerDNS API manually
curl -H "X-API-Key: your-api-key" http://powerdns-server:8081/api/v1/servers/localhost/zones

# Check DNS resolution
nslookup cluster.example.com
dig A cluster.example.com
dig AAAA cluster.example.com
```

## Documentation

For additional detailed documentation, see the `docs/` directory:

- **[GitHub Actions CI/CD](docs/github-actions.md)**: Complete guide to the automated build and publish pipeline
- **[PowerDNS Setup](examples/powerdns-setup.md)**: PowerDNS server configuration examples  
- **[go-powerdns Usage](examples/go-powerdns-usage.md)**: Detailed examples of the go-powerdns library
- **[Node Setup](examples/node-setup.sh)**: Script to configure k3s node annotations

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

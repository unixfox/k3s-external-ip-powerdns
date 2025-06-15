# PowerDNS Setup Example

This guide shows how to set up PowerDNS with API support for use with the k8s-external-ip-powerdns sync service.

## PowerDNS Configuration

### 1. Install PowerDNS Authoritative Server

```bash
# Ubuntu/Debian
apt-get update
apt-get install pdns-server pdns-backend-sqlite3

# CentOS/RHEL
yum install pdns pdns-backend-sqlite
```

### 2. Configure PowerDNS

Edit `/etc/powerdns/pdns.conf`:

```ini
# Basic configuration
local-address=0.0.0.0
local-port=53

# Database backend (SQLite example)
launch=gsqlite3
gsqlite3-database=/var/lib/powerdns/pdns.sqlite3

# API configuration
api=yes
api-key=your-secret-api-key-here
webserver=yes
webserver-address=0.0.0.0
webserver-port=8081
webserver-allow-from=0.0.0.0/0

# Logging
log-dns-queries=yes
log-dns-details=yes
loglevel=4
```

### 3. Initialize Database

```bash
# Create the database schema
sudo -u pdns pdnsutil create-bind-db /var/lib/powerdns/pdns.sqlite3
```

### 4. Create DNS Zone

```bash
# Create a new zone
pdnsutil create-zone example.com

# Set zone kind to Native
pdnsutil set-kind example.com Native

# Add nameserver records
pdnsutil add-record example.com @ NS ns1.example.com
pdnsutil add-record example.com @ NS ns2.example.com

# Add SOA record (if not automatically created)
pdnsutil add-record example.com @ SOA "ns1.example.com admin.example.com 1 3600 1800 1209600 300"

# The sync service will manage the cluster record automatically
# pdnsutil add-record example.com cluster A 192.168.1.1
```

### 5. Start PowerDNS

```bash
systemctl enable pdns
systemctl start pdns
```

## Docker Compose Example

Here's a complete Docker Compose setup for PowerDNS:

```yaml
version: '3.8'

services:
  powerdns:
    image: powerdns/pdns-auth-48:latest
    container_name: powerdns
    environment:
      PDNS_AUTH_API_KEY: your-secret-api-key-here
      PDNS_AUTH_WEBSERVER: 'yes'
      PDNS_AUTH_WEBSERVER_ADDRESS: '0.0.0.0'
      PDNS_AUTH_WEBSERVER_PORT: '8081'
      PDNS_AUTH_WEBSERVER_ALLOW_FROM: '0.0.0.0/0'
      PDNS_AUTH_API: 'yes'
      PDNS_AUTH_LOCAL_ADDRESS: '0.0.0.0'
      PDNS_AUTH_LOCAL_PORT: '53'
      PDNS_AUTH_LAUNCH: 'gsqlite3'
      PDNS_AUTH_GSQLITE3_DATABASE: '/var/lib/powerdns/pdns.sqlite3'
    volumes:
      - powerdns-data:/var/lib/powerdns
    ports:
      - "53:53/udp"
      - "53:53/tcp"
      - "8081:8081"
    restart: unless-stopped

  powerdns-admin:
    image: ngoduykhanh/powerdns-admin:latest
    container_name: powerdns-admin
    environment:
      SECRET_KEY: 'your-secret-key-for-admin'
      SQLALCHEMY_DATABASE_URI: 'sqlite:////data/pda.db'
      PDNS_STATS_URL: 'http://powerdns:8081/'
      PDNS_API_KEY: 'your-secret-api-key-here'
      PDNS_VERSION: '4.8'
    volumes:
      - powerdns-admin-data:/data
    ports:
      - "9191:80"
    depends_on:
      - powerdns
    restart: unless-stopped

volumes:
  powerdns-data:
  powerdns-admin-data:
```

## API Testing

Test the PowerDNS API to ensure it's working:

```bash
# List all zones
curl -H "X-API-Key: your-secret-api-key-here" \
     http://localhost:8081/api/v1/servers/localhost/zones

# Get zone details
curl -H "X-API-Key: your-secret-api-key-here" \
     http://localhost:8081/api/v1/servers/localhost/zones/example.com.

# Create a test record
curl -X PATCH \
     -H "X-API-Key: your-secret-api-key-here" \
     -H "Content-Type: application/json" \
     -d '{
       "rrsets": [
         {
           "name": "test.example.com.",
           "type": "A",
           "ttl": 300,
           "changetype": "REPLACE",
           "records": [
             {"content": "192.168.1.1", "disabled": false}
           ]
         }
       ]
     }' \
     http://localhost:8081/api/v1/servers/localhost/zones/example.com.
```

## DNS Testing

Test DNS resolution:

```bash
# Test with dig
dig @localhost test.example.com A
dig @localhost test.example.com AAAA

# Test with nslookup
nslookup test.example.com localhost
```

## Security Considerations

1. **API Key Security**: Use a strong, randomly generated API key
2. **Access Control**: Restrict API access to trusted networks only
3. **Firewall**: Configure firewall rules for DNS (port 53) and API (port 8081)
4. **SSL/TLS**: Consider using HTTPS for the API in production

## Troubleshooting

### Common Issues

1. **API not responding**: Check if webserver is enabled and port is accessible
2. **Zone not found**: Verify zone exists and is properly configured
3. **Permission denied**: Check file permissions on database and log files
4. **DNS queries failing**: Verify PowerDNS is listening on the correct interface

### Log Locations

- PowerDNS logs: `/var/log/pdns.log` or journalctl for systemd
- Database: `/var/lib/powerdns/pdns.sqlite3`
- Configuration: `/etc/powerdns/pdns.conf`

### Useful Commands

```bash
# Check PowerDNS status
systemctl status pdns

# View logs
journalctl -u pdns -f

# Test configuration
pdns_server --config-check

# List zones
pdnsutil list-all-zones

# Show zone records
pdnsutil list-zone example.com
```

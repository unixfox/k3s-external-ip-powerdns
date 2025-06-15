# Using the go-powerdns Library

This document explains how the k8s-external-ip-powerdns application uses the official `go-powerdns` library for PowerDNS integration.

## Library Benefits

The `go-powerdns` library provides several advantages over a custom HTTP client:

### 🔒 **Type Safety**
- Strongly typed API calls with compile-time error checking
- Proper Go structs for all PowerDNS API objects
- Enum-like constants for record types (`powerdns.RRTypeA`, `powerdns.RRTypeAAAA`)

### 🚀 **Simplified API**
- High-level methods like `Records.Change()` and `Records.Delete()`
- Automatic handling of HTTP details (headers, status codes, etc.)
- Built-in retry logic and error handling

### 🔧 **Context Support**
- Full Go context support for cancellation and timeouts
- Proper resource cleanup and connection management
- Graceful handling of long-running operations

### 📝 **Better Error Handling**
- Detailed error messages with context
- Proper HTTP status code interpretation
- PowerDNS-specific error handling

## Implementation Details

### Client Initialization

```go
// Initialize the PowerDNS client with custom HTTP client
pdns := powerdns.New(
    config.PowerDNSURL,     // "http://powerdns-server:8081"
    config.PowerDNSVHost,   // "localhost" (virtual host)
    powerdns.WithAPIKey(config.PowerDNSAPIKey), // API key option
    powerdns.WithHTTPClient(&http.Client{       // Custom HTTP client
        Timeout: 30 * time.Second,
    }),
)
```

### Connection Testing

```go
// Test connection and verify server availability
ctx := context.Background()
servers, err := pdns.Servers.List(ctx)
if err != nil {
    log.Fatalf("Failed to connect to PowerDNS API: %v", err)
}
log.Printf("Connected to PowerDNS API, found %d servers", len(servers))
```

### Zone Verification

```go
// Verify that the target DNS zone exists
_, err = pdns.Zones.Get(ctx, config.DNSZone)
if err != nil {
    log.Fatalf("Failed to access DNS zone %s: %v", config.DNSZone, err)
}
```

### Record Management

#### Creating/Updating A Records (IPv4)

```go
// Update A record with multiple IPv4 addresses
ipv4Records := []string{"192.168.1.1", "192.168.1.2"}
err := pdns.Records.Change(
    ctx,                    // Context for cancellation
    "example.com.",         // Zone name (must end with .)
    "cluster.example.com.", // Record name (FQDN)
    powerdns.RRTypeA,       // Record type constant
    uint32(300),            // TTL in seconds
    ipv4Records,            // Array of IP addresses
)
```

#### Creating/Updating AAAA Records (IPv6)

```go
// Update AAAA record with IPv6 addresses
ipv6Records := []string{"2001:db8::1", "2001:db8::2"}
err := pdns.Records.Change(
    ctx,
    "example.com.",
    "cluster.example.com.",
    powerdns.RRTypeAAAA,    // IPv6 record type
    uint32(300),
    ipv6Records,
)
```

#### Deleting Records

```go
// Delete A record when no IPv4 addresses exist
err := pdns.Records.Delete(
    ctx,
    "example.com.",
    "cluster.example.com.",
    powerdns.RRTypeA,
)

// Delete AAAA record when no IPv6 addresses exist
err := pdns.Records.Delete(
    ctx,
    "example.com.",
    "cluster.example.com.",
    powerdns.RRTypeAAAA,
)
```

### Error Handling

The library provides detailed error information:

```go
err := pdns.Records.Change(ctx, zone, record, recordType, ttl, addresses)
if err != nil {
    // The error contains context about what failed
    return fmt.Errorf("failed to update %s record: %w", recordType, err)
}
```

Common error scenarios:
- **Network errors**: Connection timeouts, DNS resolution failures
- **Authentication errors**: Invalid API key or unauthorized access
- **Zone errors**: Zone doesn't exist or is not accessible
- **Record errors**: Invalid record format or conflicting records

### Record Type Constants

The library provides constants for all PowerDNS record types:

```go
powerdns.RRTypeA      // "A" - IPv4 addresses
powerdns.RRTypeAAAA   // "AAAA" - IPv6 addresses  
powerdns.RRTypeCNAME  // "CNAME" - Canonical name
powerdns.RRTypeMX     // "MX" - Mail exchange
powerdns.RRTypeTXT    // "TXT" - Text records
powerdns.RRTypeNS     // "NS" - Name server
powerdns.RRTypeSOA    // "SOA" - Start of authority
// ... and many more
```

## Complete Example

Here's a simplified version of how our application uses the library:

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/joeig/go-powerdns/v3"
)

func updateDNSRecords(ctx context.Context, pdns *powerdns.Client, 
                     zone, record string, ipv4s, ipv6s []string) error {
    
    // Update IPv4 A records
    if len(ipv4s) > 0 {
        err := pdns.Records.Change(ctx, zone, record, 
            powerdns.RRTypeA, 300, ipv4s)
        if err != nil {
            return fmt.Errorf("failed to update A record: %w", err)
        }
        log.Printf("Updated A record with %d IPv4 addresses", len(ipv4s))
    } else {
        // Clean up existing A records
        pdns.Records.Delete(ctx, zone, record, powerdns.RRTypeA)
    }
    
    // Update IPv6 AAAA records  
    if len(ipv6s) > 0 {
        err := pdns.Records.Change(ctx, zone, record,
            powerdns.RRTypeAAAA, 300, ipv6s)
        if err != nil {
            return fmt.Errorf("failed to update AAAA record: %w", err)
        }
        log.Printf("Updated AAAA record with %d IPv6 addresses", len(ipv6s))
    } else {
        // Clean up existing AAAA records
        pdns.Records.Delete(ctx, zone, record, powerdns.RRTypeAAAA)
    }
    
    return nil
}

func main() {
    // Initialize PowerDNS client
    pdns := powerdns.New(
        "http://powerdns-server:8081",
        "localhost",
        powerdns.WithAPIKey("your-api-key"),
    )
    
    ctx := context.Background()
    
    // Test connection
    if _, err := pdns.Servers.List(ctx); err != nil {
        log.Fatal("PowerDNS connection failed:", err)
    }
    
    // Update records
    ipv4s := []string{"192.168.1.1", "192.168.1.2"}
    ipv6s := []string{"2001:db8::1", "2001:db8::2"}
    
    err := updateDNSRecords(ctx, pdns, "example.com.", 
        "cluster.example.com.", ipv4s, ipv6s)
    if err != nil {
        log.Fatal("DNS update failed:", err)
    }
    
    log.Println("DNS records updated successfully!")
}
```

## Library Documentation

For complete documentation of the go-powerdns library, see:
- **GitHub**: https://github.com/joeig/go-powerdns
- **GoDoc**: https://pkg.go.dev/github.com/joeig/go-powerdns/v3
- **Examples**: https://pkg.go.dev/github.com/joeig/go-powerdns/v3#pkg-examples

## Supported PowerDNS Versions

The go-powerdns library supports:
- PowerDNS Authoritative Server 4.7+
- PowerDNS Authoritative Server 4.8+  
- PowerDNS Authoritative Server 4.9+

## Migration from Custom HTTP Client

If you were previously using a custom HTTP client for PowerDNS, the go-powerdns library provides these advantages:

| Custom HTTP Client | go-powerdns Library |
|-------------------|-------------------|
| Manual JSON marshaling | Automatic type handling |
| Custom error parsing | Built-in error types |
| Manual HTTP headers | Automatic authentication |
| No type safety | Compile-time type checking |
| Manual retry logic | Built-in resilience |
| No context support | Full context integration |

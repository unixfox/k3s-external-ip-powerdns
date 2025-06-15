package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/joeig/go-powerdns/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Build information set via ldflags
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

const (
	ExternalIPAnnotation = "k3s.io/external-ip"
	DefaultSyncInterval  = 30 * time.Second
	DefaultTTL          = 300
)

type Config struct {
	PowerDNSURL     string
	PowerDNSAPIKey  string
	PowerDNSVHost   string
	DNSZone         string
	DNSRecord       string
	SyncInterval    time.Duration
	KubeConfig      string
	TTL             int
}

type IPAddress struct {
	IP      net.IP
	IsIPv6  bool
	String  string
}

func parseIPAddresses(ipString string) ([]IPAddress, error) {
	if ipString == "" {
		return nil, nil
	}

	var addresses []IPAddress
	ipStrings := strings.Split(ipString, ",")

	for _, ipStr := range ipStrings {
		ipStr = strings.TrimSpace(ipStr)
		if ipStr == "" {
			continue
		}

		ip := net.ParseIP(ipStr)
		if ip == nil {
			log.Printf("Warning: invalid IP address format: %s", ipStr)
			continue
		}

		isIPv6 := ip.To4() == nil
		addresses = append(addresses, IPAddress{
			IP:     ip,
			IsIPv6: isIPv6,
			String: ipStr,
		})
	}

	return addresses, nil
}

func validateDNSZone(zone string) string {
	// Ensure zone ends with a dot (FQDN)
	if !strings.HasSuffix(zone, ".") {
		zone += "."
	}
	return zone
}

func validateDNSRecord(record string) string {
	// Ensure record name ends with a dot (FQDN)
	if !strings.HasSuffix(record, ".") {
		record += "."
	}
	return record
}

func getKubernetesClient(kubeConfig string) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	if kubeConfig != "" {
		// Use provided kubeconfig file
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	} else {
		// Try in-cluster config first
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig location
			kubeConfigPath := os.Getenv("KUBECONFIG")
			if kubeConfigPath == "" {
				homeDir, _ := os.UserHomeDir()
				kubeConfigPath = fmt.Sprintf("%s/.kube/config", homeDir)
			}
			config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

func fetchExternalIPs(clientset *kubernetes.Clientset) ([]IPAddress, error) {
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "forbidden") {
			return nil, fmt.Errorf("failed to list nodes due to insufficient permissions: %w\n\nThis error indicates the service account lacks proper RBAC permissions.\nPlease ensure the service account has the following permissions:\n- apiGroups: [\"\"]\n  resources: [\"nodes\"]\n  verbs: [\"get\", \"list\", \"watch\"]\n\nSee k8s-deployment.yaml for the complete RBAC configuration.", err)
		}
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var allIPs []IPAddress
	seenIPs := make(map[string]bool)

	for _, node := range nodes.Items {
		externalIPAnnotation, exists := node.Annotations[ExternalIPAnnotation]
		if !exists || externalIPAnnotation == "" {
			log.Printf("Node %s does not have external IP annotation", node.Name)
			continue
		}

		log.Printf("Found external IPs for node %s: %s", node.Name, externalIPAnnotation)

		ips, err := parseIPAddresses(externalIPAnnotation)
		if err != nil {
			log.Printf("Error parsing IPs for node %s: %v", node.Name, err)
			continue
		}

		for _, ip := range ips {
			// Deduplicate IPs
			if !seenIPs[ip.String] {
				seenIPs[ip.String] = true
				allIPs = append(allIPs, ip)
			}
		}
	}

	// Sort IPs for consistent ordering (IPv4 first, then IPv6)
	sort.Slice(allIPs, func(i, j int) bool {
		if allIPs[i].IsIPv6 != allIPs[j].IsIPv6 {
			return !allIPs[i].IsIPv6 // IPv4 (false) comes before IPv6 (true)
		}
		return allIPs[i].String < allIPs[j].String
	})

	return allIPs, nil
}

func updateDNSRecords(ctx context.Context, pdns *powerdns.Client, config *Config, ipAddresses []IPAddress) error {
	// Group IP addresses by type
	var ipv4Records []string
	var ipv6Records []string

	for _, ip := range ipAddresses {
		if ip.IsIPv6 {
			ipv6Records = append(ipv6Records, ip.String)
		} else {
			ipv4Records = append(ipv4Records, ip.String)
		}
	}

	// Validate and ensure proper FQDN format
	recordName := validateDNSRecord(config.DNSRecord)
	zone := validateDNSZone(config.DNSZone)

	// Update A records for IPv4
	if len(ipv4Records) > 0 {
		log.Printf("Updating A record for %s with %d IPv4 addresses", recordName, len(ipv4Records))
		err := pdns.Records.Change(ctx, zone, recordName, powerdns.RRTypeA, uint32(config.TTL), ipv4Records)
		if err != nil {
			return fmt.Errorf("failed to update A record: %w", err)
		}
		log.Printf("Successfully updated A record for %s", recordName)
	} else {
		// Delete existing A records if no IPv4 addresses
		log.Printf("No IPv4 addresses found, deleting A record for %s", recordName)
		err := pdns.Records.Delete(ctx, zone, recordName, powerdns.RRTypeA)
		if err != nil {
			// Check if it's a "not found" error and log accordingly
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
				log.Printf("A record for %s does not exist (already deleted)", recordName)
			} else {
				log.Printf("Warning: failed to delete A record: %v", err)
			}
		}
	}

	// Update AAAA records for IPv6
	if len(ipv6Records) > 0 {
		log.Printf("Updating AAAA record for %s with %d IPv6 addresses", recordName, len(ipv6Records))
		err := pdns.Records.Change(ctx, zone, recordName, powerdns.RRTypeAAAA, uint32(config.TTL), ipv6Records)
		if err != nil {
			return fmt.Errorf("failed to update AAAA record: %w", err)
		}
		log.Printf("Successfully updated AAAA record for %s", recordName)
	} else {
		// Delete existing AAAA records if no IPv6 addresses
		log.Printf("No IPv6 addresses found, deleting AAAA record for %s", recordName)
		err := pdns.Records.Delete(ctx, zone, recordName, powerdns.RRTypeAAAA)
		if err != nil {
			// Check if it's a "not found" error and log accordingly
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
				log.Printf("AAAA record for %s does not exist (already deleted)", recordName)
			} else {
				log.Printf("Warning: failed to delete AAAA record: %v", err)
			}
		}
	}

	return nil
}

func loadConfig() (*Config, error) {
	config := &Config{
		SyncInterval: DefaultSyncInterval,
		TTL:          DefaultTTL,
	}

	if url := os.Getenv("POWERDNS_URL"); url != "" {
		config.PowerDNSURL = url
	} else {
		return nil, fmt.Errorf("POWERDNS_URL environment variable is required")
	}

	if apiKey := os.Getenv("POWERDNS_API_KEY"); apiKey != "" {
		config.PowerDNSAPIKey = apiKey
	} else {
		return nil, fmt.Errorf("POWERDNS_API_KEY environment variable is required")
	}

	if vhost := os.Getenv("POWERDNS_VHOST"); vhost != "" {
		config.PowerDNSVHost = vhost
	} else {
		// Default to localhost if not specified
		config.PowerDNSVHost = "localhost"
	}

	if zone := os.Getenv("DNS_ZONE"); zone != "" {
		config.DNSZone = validateDNSZone(zone)
	} else {
		return nil, fmt.Errorf("DNS_ZONE environment variable is required")
	}

	if record := os.Getenv("DNS_RECORD"); record != "" {
		config.DNSRecord = validateDNSRecord(record)
	} else {
		return nil, fmt.Errorf("DNS_RECORD environment variable is required")
	}

	if interval := os.Getenv("SYNC_INTERVAL"); interval != "" {
		if duration, err := time.ParseDuration(interval); err == nil {
			config.SyncInterval = duration
		} else {
			log.Printf("Warning: invalid SYNC_INTERVAL format, using default: %v", DefaultSyncInterval)
		}
	}

	if ttlStr := os.Getenv("DNS_TTL"); ttlStr != "" {
		if ttl, err := time.ParseDuration(ttlStr); err == nil {
			config.TTL = int(ttl.Seconds())
		} else {
			log.Printf("Warning: invalid DNS_TTL format, using default: %d seconds", DefaultTTL)
		}
	}

	config.KubeConfig = os.Getenv("KUBECONFIG")

	return config, nil
}

func syncDNSRecords(ctx context.Context, clientset *kubernetes.Clientset, pdns *powerdns.Client, config *Config) error {
	log.Println("Fetching external IP addresses from Kubernetes nodes...")

	ips, err := fetchExternalIPs(clientset)
	if err != nil {
		return fmt.Errorf("failed to fetch external IPs: %w", err)
	}

	if len(ips) == 0 {
		log.Println("No external IP addresses found")
		// Still try to clean up existing records
		return updateDNSRecords(ctx, pdns, config, ips)
	}

	log.Printf("Found %d external IP addresses:", len(ips))
	for _, ip := range ips {
		ipType := "IPv4"
		if ip.IsIPv6 {
			ipType = "IPv6"
		}
		log.Printf("  %s (%s)", ip.String, ipType)
	}

	log.Printf("Updating DNS records for %s in zone %s...", config.DNSRecord, config.DNSZone)

	err = updateDNSRecords(ctx, pdns, config, ips)
	if err != nil {
		return fmt.Errorf("failed to update DNS records: %w", err)
	}

	return nil
}

func main() {
	log.Printf("Starting k8s-external-ip-powerdns sync service...")
	log.Printf("Version: %s, Commit: %s, Build Date: %s", version, commit, buildDate)

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded:")
	log.Printf("  PowerDNS URL: %s", config.PowerDNSURL)
	log.Printf("  PowerDNS VHost: %s", config.PowerDNSVHost)
	log.Printf("  DNS Zone: %s", config.DNSZone)
	log.Printf("  DNS Record: %s", config.DNSRecord)
	log.Printf("  DNS TTL: %d seconds", config.TTL)
	log.Printf("  Sync Interval: %v", config.SyncInterval)

	clientset, err := getKubernetesClient(config.KubeConfig)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Test Kubernetes permissions before starting
	ctx := context.Background()
	log.Println("Verifying Kubernetes permissions...")
	_, err = clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		log.Fatalf("Failed to access Kubernetes nodes - check service account permissions: %v\n\nRequired RBAC permissions:\n- apiGroups: [\"\"]\n  resources: [\"nodes\"]\n  verbs: [\"get\", \"list\", \"watch\"]\n\nSee k8s-deployment.yaml for proper RBAC configuration.", err)
	}
	log.Println("Kubernetes permissions verified successfully")

	// Initialize PowerDNS client with proper options
	pdns := powerdns.New(
		config.PowerDNSURL,
		config.PowerDNSVHost,
		powerdns.WithAPIKey(config.PowerDNSAPIKey),
		powerdns.WithHTTPClient(&http.Client{
			Timeout: 30 * time.Second,
		}),
	)

	// Test PowerDNS connection
	servers, err := pdns.Servers.List(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to PowerDNS API: %v", err)
	}
	log.Printf("Connected to PowerDNS API, found %d servers", len(servers))

	// Verify zone exists
	_, err = pdns.Zones.Get(ctx, config.DNSZone)
	if err != nil {
		log.Fatalf("Failed to access DNS zone %s: %v", config.DNSZone, err)
	}
	log.Printf("Successfully verified DNS zone: %s", config.DNSZone)

	// Perform initial sync
	log.Println("Performing initial DNS sync...")
	if err := syncDNSRecords(ctx, clientset, pdns, config); err != nil {
		log.Fatalf("Initial sync failed: %v", err)
	}
	log.Println("Initial sync completed successfully")

	// Set up periodic sync
	ticker := time.NewTicker(config.SyncInterval)
	defer ticker.Stop()

	log.Printf("Starting periodic sync every %v...", config.SyncInterval)

	for {
		select {
		case <-ticker.C:
			if err := syncDNSRecords(ctx, clientset, pdns, config); err != nil {
				log.Printf("Sync failed: %v", err)
			}
		}
	}
}

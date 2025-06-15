package main

import (
	"net"
	"strings"
	"testing"
)

func TestParseIPAddresses(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int
		ipv4Count int
		ipv6Count int
	}{
		{
			name:      "Single IPv4",
			input:     "152.67.73.95",
			expected:  1,
			ipv4Count: 1,
			ipv6Count: 0,
		},
		{
			name:      "Single IPv6",
			input:     "2603:c022:5:1e00:a452:9f75:7f83:3a88",
			expected:  1,
			ipv4Count: 0,
			ipv6Count: 1,
		},
		{
			name:      "Mixed IPv4 and IPv6",
			input:     "152.67.73.95,2603:c022:5:1e00:a452:9f75:7f83:3a88",
			expected:  2,
			ipv4Count: 1,
			ipv6Count: 1,
		},
		{
			name:      "Multiple IPv4",
			input:     "192.168.1.1,10.0.0.1",
			expected:  2,
			ipv4Count: 2,
			ipv6Count: 0,
		},
		{
			name:      "With spaces",
			input:     "152.67.73.95, 2603:c022:5:1e00:a452:9f75:7f83:3a88",
			expected:  2,
			ipv4Count: 1,
			ipv6Count: 1,
		},
		{
			name:      "Empty string",
			input:     "",
			expected:  0,
			ipv4Count: 0,
			ipv6Count: 0,
		},
		{
			name:      "Invalid IP",
			input:     "invalid-ip",
			expected:  0,
			ipv4Count: 0,
			ipv6Count: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseIPAddresses(tt.input)
			if err != nil {
				t.Errorf("parseIPAddresses() error = %v", err)
				return
			}

			if len(result) != tt.expected {
				t.Errorf("parseIPAddresses() = %v, want %v addresses", len(result), tt.expected)
			}

			ipv4Count := 0
			ipv6Count := 0
			for _, ip := range result {
				if ip.IsIPv6 {
					ipv6Count++
				} else {
					ipv4Count++
				}
			}

			if ipv4Count != tt.ipv4Count {
				t.Errorf("parseIPAddresses() IPv4 count = %v, want %v", ipv4Count, tt.ipv4Count)
			}

			if ipv6Count != tt.ipv6Count {
				t.Errorf("parseIPAddresses() IPv6 count = %v, want %v", ipv6Count, tt.ipv6Count)
			}
		})
	}
}

func TestIPAddressClassification(t *testing.T) {
	tests := []struct {
		ip     string
		isIPv6 bool
	}{
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"172.16.0.1", false},
		{"8.8.8.8", false},
		{"::1", true},
		{"2001:db8::1", true},
		{"fe80::1", true},
		{"2603:c022:5:1e00:a452:9f75:7f83:3a88", true},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			parsedIP := net.ParseIP(tt.ip)
			if parsedIP == nil {
				t.Errorf("Failed to parse IP: %s", tt.ip)
				return
			}

			isIPv6 := parsedIP.To4() == nil
			if isIPv6 != tt.isIPv6 {
				t.Errorf("IP %s: expected IPv6=%v, got IPv6=%v", tt.ip, tt.isIPv6, isIPv6)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	// Test that FQDN handling works correctly
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Domain without trailing dot",
			input:    "cluster.example.com",
			expected: "cluster.example.com.",
		},
		{
			name:     "Domain with trailing dot",
			input:    "cluster.example.com.",
			expected: "cluster.example.com.",
		},
		{
			name:     "Subdomain without trailing dot",
			input:    "api.cluster.example.com",
			expected: "api.cluster.example.com.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recordName := tc.input
			if !strings.HasSuffix(recordName, ".") {
				recordName += "."
			}
			
			if recordName != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, recordName)
			}
		})
	}
}

func TestDNSValidationFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Zone without trailing dot",
			input:    "example.com",
			expected: "example.com.",
		},
		{
			name:     "Zone with trailing dot",
			input:    "example.com.",
			expected: "example.com.",
		},
		{
			name:     "Record without trailing dot",
			input:    "cluster.example.com",
			expected: "cluster.example.com.",
		},
		{
			name:     "Record with trailing dot",
			input:    "cluster.example.com.",
			expected: "cluster.example.com.",
		},
	}

	for _, tt := range tests {
		t.Run("Zone_"+tt.name, func(t *testing.T) {
			result := validateDNSZone(tt.input)
			if result != tt.expected {
				t.Errorf("validateDNSZone(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})

		t.Run("Record_"+tt.name, func(t *testing.T) {
			result := validateDNSRecord(tt.input)
			if result != tt.expected {
				t.Errorf("validateDNSRecord(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

# Example: Adding external IP annotation and labels to Kubernetes nodes

# This file demonstrates how to add the k3s.io/external-ip annotation
# and optional labels to a Kubernetes node for the sync service.

# Step 1: Add external IP annotation
# Method 1: Using kubectl patch
kubectl patch node <node-name> -p '{"metadata":{"annotations":{"k3s.io/external-ip":"152.67.73.95,2603:c022:5:1e00:a452:9f75:7f83:3a88"}}}'

# Method 2: Using kubectl annotate
kubectl annotate node <node-name> k3s.io/external-ip="152.67.73.95,2603:c022:5:1e00:a452:9f75:7f83:3a88"

# Step 2: Add labels for node selection (optional)
# Only nodes with matching labels will be included if NODE_SELECTOR is configured

# Example: Label node for DNS synchronization
kubectl label node <node-name> dns-sync=enabled

# Example: Label with role and environment
kubectl label node <node-name> role=worker environment=production

# Example: Complete setup with both annotation and label
kubectl annotate node node1 k3s.io/external-ip="192.168.1.100"
kubectl label node node1 dns-sync=enabled

# Method 3: Using a YAML manifest for complete node configuration
# Save the following as node-patch.yaml and apply with: kubectl apply -f node-patch.yaml

apiVersion: v1
kind: Node
metadata:
  name: <node-name>
  labels:
    dns-sync: enabled
    role: worker
    environment: production
  annotations:
    k3s.io/external-ip: "152.67.73.95,2603:c022:5:1e00:a452:9f75:7f83:3a88"

# Example with multiple nodes:

# Node 1 - IPv4 only with DNS sync enabled
kubectl annotate node node1 k3s.io/external-ip="192.168.1.100"
kubectl label node node1 dns-sync=enabled

# Node 2 - IPv6 only with DNS sync enabled
kubectl annotate node node2 k3s.io/external-ip="2001:db8::1"
kubectl label node node2 dns-sync=enabled

# Node 3 - Both IPv4 and IPv6 with DNS sync enabled
kubectl annotate node node3 k3s.io/external-ip="10.0.0.1,2001:db8::2"
kubectl label node node3 dns-sync=enabled

# Node 4 - Has external IP but NOT included in DNS (no label)
kubectl annotate node node4 k3s.io/external-ip="10.0.0.2"
# This node will be ignored if NODE_SELECTOR=dns-sync=enabled

# Node 2 - IPv6 only  
kubectl annotate node node2 k3s.io/external-ip="2001:db8::1"

# Node 3 - Both IPv4 and IPv6
kubectl annotate node node3 k3s.io/external-ip="10.0.0.1,2001:db8::2"

# To remove the annotation:
kubectl annotate node <node-name> k3s.io/external-ip-

# To view current annotations:
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.k3s\.io/external-ip}{"\n"}{end}'

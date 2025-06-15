# Example: Adding external IP annotation to a Kubernetes node

# This file demonstrates how to add the k3s.io/external-ip annotation
# to a Kubernetes node so that the sync service can detect it.

# Method 1: Using kubectl patch
kubectl patch node <node-name> -p '{"metadata":{"annotations":{"k3s.io/external-ip":"152.67.73.95,2603:c022:5:1e00:a452:9f75:7f83:3a88"}}}'

# Method 2: Using kubectl annotate
kubectl annotate node <node-name> k3s.io/external-ip="152.67.73.95,2603:c022:5:1e00:a452:9f75:7f83:3a88"

# Method 3: Using a YAML manifest
# Save the following as node-patch.yaml and apply with: kubectl apply -f node-patch.yaml

apiVersion: v1
kind: Node
metadata:
  name: <node-name>
  annotations:
    k3s.io/external-ip: "152.67.73.95,2603:c022:5:1e00:a452:9f75:7f83:3a88"

# Example with multiple nodes:

# Node 1 - IPv4 only
kubectl annotate node node1 k3s.io/external-ip="192.168.1.100"

# Node 2 - IPv6 only  
kubectl annotate node node2 k3s.io/external-ip="2001:db8::1"

# Node 3 - Both IPv4 and IPv6
kubectl annotate node node3 k3s.io/external-ip="10.0.0.1,2001:db8::2"

# To remove the annotation:
kubectl annotate node <node-name> k3s.io/external-ip-

# To view current annotations:
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.annotations.k3s\.io/external-ip}{"\n"}{end}'

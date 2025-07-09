# Infrastructure Alerts Runbook

## Alert Categories
1. **Node Issues**: CPU, Memory, Disk, Network
2. **Cluster Issues**: API server, etcd, scheduler
3. **Storage Issues**: PV capacity, I/O performance
4. **Network Issues**: Connectivity, DNS, Load balancers

## Node Issues

### High CPU Usage
**Alert**: Node CPU > 85% for 5 minutes

#### Diagnosis
```bash
# Check node CPU usage
kubectl top nodes
kubectl describe node <node-name> | grep -A 10 "Allocated resources"

# Find CPU-intensive pods
kubectl top pods --all-namespaces --sort-by=cpu | head -20

# Check for CPU throttling
kubectl get --raw /apis/metrics.k8s.io/v1beta1/nodes/<node-name> | jq '.usage'

# System-level investigation
ssh <node> "top -b -n 1 | head -20"
ssh <node> "ps aux | sort -nrk 3,3 | head -10"
```

#### Remediation
```bash
# Move pods to other nodes
kubectl cordon <node-name>
kubectl drain <node-name> --ignore-daemonsets

# Scale down non-critical workloads
kubectl scale deployment <high-cpu-app> --replicas=<lower-count>

# Add CPU limits to pods
kubectl patch deployment <app> -p '{"spec":{"template":{"spec":{"containers":[{"name":"<container>","resources":{"limits":{"cpu":"1"}}}]}}}}'
```

### High Memory Usage
**Alert**: Node Memory > 90% for 5 minutes

#### Diagnosis
```bash
# Check memory usage
kubectl top nodes
kubectl describe node <node-name> | grep -A 5 "memory"

# Find memory-intensive pods
kubectl top pods --all-namespaces --sort-by=memory | head -20

# Check for OOM kills
kubectl get events --all-namespaces | grep OOMKilled
journalctl -u kubelet | grep -i "out of memory"

# Detailed memory analysis
ssh <node> "free -h"
ssh <node> "ps aux | sort -nrk 4,4 | head -10"
ssh <node> "cat /proc/meminfo | grep -E 'MemTotal|MemFree|MemAvailable|Cached|Buffers'"
```

#### Remediation
```bash
# Evict pods from memory-pressured node
kubectl drain <node-name> --ignore-daemonsets --delete-emptydir-data

# Set memory limits
kubectl patch deployment <app> -p '{"spec":{"template":{"spec":{"containers":[{"name":"<container>","resources":{"limits":{"memory":"2Gi"},"requests":{"memory":"1Gi"}}}]}}}}'

# Clear page cache (temporary relief)
ssh <node> "sync && echo 1 > /proc/sys/vm/drop_caches"

# Enable memory overcommit (use with caution)
ssh <node> "echo 1 > /proc/sys/vm/overcommit_memory"
```

## Disk Space Issues

### Node Disk Space Low
**Alert**: Node disk usage > 85%

#### Diagnosis
```bash
# Check disk usage across nodes
kubectl get nodes -o custom-columns=NAME:.metadata.name,DISK:.status.allocatable.ephemeral-storage

# Find large directories
ssh <node> "df -h"
ssh <node> "du -sh /var/lib/docker/* | sort -hr | head -10"
ssh <node> "du -sh /var/lib/kubelet/* | sort -hr | head -10"

# Check for large container logs
kubectl logs --all-namespaces --selector="" --tail=1 2>&1 | grep "^Error" | grep "too large"

# Find pods with high disk usage
for pod in $(kubectl get pods --all-namespaces -o json | jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name)"'); do
  echo "Checking $pod"
  kubectl exec -n ${pod%/*} ${pod#*/} -- df -h 2>/dev/null | grep -E "^/dev|^overlay"
done
```

#### Remediation
```bash
# Clean up Docker resources
ssh <node> "docker system prune -af --volumes"

# Clean up old images
ssh <node> "crictl rmi --prune"

# Rotate and compress logs
ssh <node> "journalctl --vacuum-size=1G"
ssh <node> "find /var/log -name '*.log' -size +100M -exec gzip {} \;"

# Delete old pods and containers
kubectl delete pods --all-namespaces --field-selector status.phase=Failed
kubectl delete pods --all-namespaces --field-selector status.phase=Succeeded

# Increase volume size (if using cloud storage)
kubectl patch pvc <pvc-name> -p '{"spec":{"resources":{"requests":{"storage":"100Gi"}}}}'
```

### PersistentVolume Full
**Alert**: PV usage > 90%

#### Diagnosis
```bash
# Check PV usage
kubectl get pv -o custom-columns=NAME:.metadata.name,CAPACITY:.spec.capacity.storage,CLAIM:.spec.claimRef.name

# Check PVC usage
kubectl exec -n <namespace> <pod> -- df -h | grep "/data"

# Find large files
kubectl exec -n <namespace> <pod> -- find /data -size +1G -type f -exec ls -lh {} \;
```

#### Remediation
```bash
# Resize PVC (if storage class supports it)
kubectl patch pvc <pvc-name> -n <namespace> -p '{"spec":{"resources":{"requests":{"storage":"200Gi"}}}}'

# Clean up old data
kubectl exec -n <namespace> <pod> -- find /data -mtime +30 -type f -delete

# Compress large files
kubectl exec -n <namespace> <pod> -- gzip /data/large-file.log

# Move data to object storage
kubectl exec -n <namespace> <pod> -- aws s3 cp /data/archive/ s3://bucket/archive/ --recursive
```

## Memory Pressure

### Node Memory Pressure
**Alert**: Node under memory pressure

#### Diagnosis
```bash
# Check memory pressure condition
kubectl describe node <node-name> | grep -A 5 "MemoryPressure"

# Check kubelet memory thresholds
ssh <node> "cat /var/lib/kubelet/config.yaml | grep -A 5 'eviction'"

# Monitor memory metrics
watch -n 5 "kubectl top nodes && echo '---' && kubectl get nodes -o custom-columns=NAME:.metadata.name,STATUS:.status.conditions[?(@.type=='MemoryPressure')].status"
```

#### Remediation
```bash
# Adjust eviction thresholds
ssh <node> "cat > /var/lib/kubelet/config.yaml << EOF
evictionHard:
  memory.available: '500Mi'
  nodefs.available: '10%'
evictionSoft:
  memory.available: '1Gi'
  nodefs.available: '15%'
evictionSoftGracePeriod:
  memory.available: '2m'
  nodefs.available: '2m'
EOF"

# Restart kubelet
ssh <node> "systemctl restart kubelet"

# Set resource limits on all pods
kubectl get deployments --all-namespaces -o json | jq -r '.items[] | "\(.metadata.namespace) \(.metadata.name)"' | while read ns dep; do
  kubectl patch deployment $dep -n $ns --type='json' -p='[{"op":"add","path":"/spec/template/spec/containers/0/resources","value":{"limits":{"memory":"1Gi"},"requests":{"memory":"512Mi"}}}]'
done
```

## Network Problems

### Network Connectivity Issues
**Alert**: Node network unreachable

#### Diagnosis
```bash
# Check node network status
kubectl get nodes -o wide
kubectl describe node <node-name> | grep -A 10 "Addresses"

# Test connectivity between nodes
kubectl run test-pod --image=busybox --rm -it -- ping <other-node-ip>

# Check CNI plugin status
kubectl get pods -n kube-system | grep -E "calico|weave|flannel|cilium"
kubectl logs -n kube-system <cni-pod>

# Network diagnostics
ssh <node> "ip addr show"
ssh <node> "ip route show"
ssh <node> "iptables -L -n -v | head -50"
```

#### Remediation
```bash
# Restart CNI pods
kubectl delete pods -n kube-system -l k8s-app=calico-node

# Reset iptables rules
ssh <node> "iptables -F && iptables -X && iptables -t nat -F && iptables -t nat -X"
ssh <node> "systemctl restart kubelet"

# Check and fix DNS
kubectl delete pods -n kube-system -l k8s-app=kube-dns
kubectl rollout restart deployment/coredns -n kube-system

# Verify network policies
kubectl get networkpolicies --all-namespaces
```

### DNS Resolution Failures
**Alert**: DNS queries failing

#### Diagnosis
```bash
# Check CoreDNS status
kubectl get pods -n kube-system -l k8s-app=kube-dns
kubectl logs -n kube-system -l k8s-app=kube-dns --tail=50

# Test DNS resolution
kubectl run -it --rm debug --image=busybox --restart=Never -- nslookup kubernetes.default
kubectl run -it --rm debug --image=tutum/dnsutils --restart=Never -- dig @10.96.0.10 kubernetes.default.svc.cluster.local

# Check DNS configuration
kubectl get configmap -n kube-system coredns -o yaml
kubectl get svc -n kube-system kube-dns
```

#### Remediation
```bash
# Restart CoreDNS
kubectl rollout restart deployment/coredns -n kube-system

# Scale CoreDNS
kubectl scale deployment/coredns -n kube-system --replicas=3

# Clear DNS cache
kubectl exec -n kube-system <coredns-pod> -- kill -SIGUSR1 1

# Update DNS config
kubectl edit configmap -n kube-system coredns
# Add: cache 30
# Add: reload
```

## Cluster Component Issues

### etcd Issues
**Alert**: etcd cluster unhealthy

#### Diagnosis
```bash
# Check etcd health
kubectl get pods -n kube-system | grep etcd
kubectl exec -n kube-system etcd-<master> -- etcdctl \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/healthcheck-client.crt \
  --key=/etc/kubernetes/pki/etcd/healthcheck-client.key \
  endpoint health

# Check etcd metrics
kubectl exec -n kube-system etcd-<master> -- etcdctl \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/healthcheck-client.crt \
  --key=/etc/kubernetes/pki/etcd/healthcheck-client.key \
  endpoint status --write-out=table
```

#### Remediation
```bash
# Defragment etcd
kubectl exec -n kube-system etcd-<master> -- etcdctl \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/healthcheck-client.crt \
  --key=/etc/kubernetes/pki/etcd/healthcheck-client.key \
  defrag

# Backup etcd
kubectl exec -n kube-system etcd-<master> -- etcdctl \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/healthcheck-client.crt \
  --key=/etc/kubernetes/pki/etcd/healthcheck-client.key \
  snapshot save /tmp/etcd-backup.db

# Increase etcd resource limits
kubectl patch pod etcd-<master> -n kube-system --type='json' -p='[{"op":"replace","path":"/spec/containers/0/resources/limits/memory","value":"4Gi"}]'
```

## Monitoring Commands

### Quick Health Check
```bash
# Cluster overview
kubectl cluster-info
kubectl get nodes
kubectl top nodes
kubectl get pods --all-namespaces | grep -v Running | grep -v Completed

# Component status
kubectl get componentstatuses
kubectl get events --all-namespaces --sort-by='.lastTimestamp' | tail -20

# Resource usage
kubectl api-resources --verbs=list --namespaced -o name | xargs -n 1 kubectl get --show-kind --ignore-not-found --all-namespaces
```

## Related Documents
- [Service Down Runbook](./service-down.md)
- [High Latency Runbook](./high-latency.md)
- [Kubernetes Troubleshooting Guide](../k8s-troubleshooting.md)
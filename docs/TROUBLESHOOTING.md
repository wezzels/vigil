# VIGIL Troubleshooting Guide

## Common Issues

### 1. Pod Won't Start

**Symptoms:**
- Pod stuck in `Pending` state
- Pod stuck in `CrashLoopBackOff`
- Pod stuck in `ImagePullBackOff`

**Diagnosis:**

```bash
# Check pod events
kubectl describe pod <pod-name> -n vigil

# Check logs
kubectl logs <pod-name> -n vigil --previous

# Check resource constraints
kubectl describe pod <pod-name> -n vigil | grep -A 10 "Conditions:"
```

**Solutions:**

| Issue | Solution |
|-------|----------|
| Insufficient resources | Scale down other pods or add nodes |
| Image not found | Check image name and registry |
| Failed mounts | Check PVC status and access |
| CrashLoopBackOff | Check application logs for errors |
| ConfigMap/Secret missing | Verify ConfigMap and Secret exist |

---

### 2. Service Unavailable

**Symptoms:**
- 503 Service Unavailable
- Connection refused
- Timeout errors

**Diagnosis:**

```bash
# Check service endpoints
kubectl get endpoints <service-name> -n vigil

# Check pod readiness
kubectl get pods -n vigil -o wide

# Test direct connection
kubectl run test --rm -it --image=busybox -- \
  wget -qO- http://<service>:8080/healthz/liveness
```

**Solutions:**

| Issue | Solution |
|-------|----------|
| No endpoints | Check pod labels match service selector |
| Readiness failing | Check health check path |
| Pod not running | See "Pod Won't Start" above |
| Network policy blocking | Check NetworkPolicy rules |

---

### 3. Database Connection Issues

**Symptoms:**
- Connection timeout
- "Too many connections"
- Authentication failed

**Diagnosis:**

```bash
# Check PostgreSQL status
kubectl exec -it postgres-0 -n vigil -- \
  psql -U vigil -c "SELECT 1;"

# Check connection count
kubectl exec -it postgres-0 -n vigil -- \
  psql -U vigil -c "SELECT count(*) FROM pg_stat_activity;"

# Check logs
kubectl logs postgres-0 -n vigil
```

**Solutions:**

| Issue | Solution |
|-------|----------|
| Too many connections | Increase max_connections or fix connection leaks |
| Authentication failed | Check credentials in Secret |
| Connection timeout | Check network policy and DNS |
| Database not ready | Wait for PostgreSQL to be ready |

---

### 4. Kafka Connection Issues

**Symptoms:**
- Producer/Consumer errors
- Timeout connecting to Kafka
- Leader not available

**Diagnosis:**

```bash
# Check Kafka brokers
kafka-broker-api-versions --bootstrap-server kafka:9092

# Check topics
kafka-topics --list --bootstrap-server kafka:9092

# Check consumer groups
kafka-consumer-groups --list --bootstrap-server kafka:9092

# Check logs
kubectl logs kafka-0 -n vigil
```

**Solutions:**

| Issue | Solution |
|-------|----------|
| No leader | Check ISR, wait for election |
| Broker down | Restart broker pod |
| Topic missing | Create topic |
| Consumer lag | Scale consumers or check for errors |

---

### 5. Redis Connection Issues

**Symptoms:**
- Cache misses
- Connection refused
- Timeout errors

**Diagnosis:**

```bash
# Check Redis
kubectl exec -it redis-0 -n vigil -- redis-cli ping

# Check memory
kubectl exec -it redis-0 -n vigil -- redis-cli info memory

# Check connections
kubectl exec -it redis-0 -n vigil -- redis-cli info clients
```

**Solutions:**

| Issue | Solution |
|-------|----------|
| Memory full | Increase memory limit or clear cache |
| Too many connections | Check connection leaks |
| Redis down | Restart Redis pod |

---

### 6. High Memory Usage

**Symptoms:**
- OOMKilled pods
- Slow performance
- Memory alerts

**Diagnosis:**

```bash
# Check pod memory
kubectl top pods -n vigil

# Get memory details
kubectl exec -it <pod-name> -n vigil -- cat /proc/meminfo

# Check heap (for Go apps)
curl http://<pod-ip>:9090/debug/heap
```

**Solutions:**

| Issue | Solution |
|-------|----------|
| Memory leak | Restart pod, investigate code |
| Insufficient limit | Increase memory limit |
| Large data sets | Implement pagination or streaming |

---

### 7. High CPU Usage

**Symptoms:**
- Slow response times
- CPU throttling
- CPU alerts

**Diagnosis:**

```bash
# Check CPU usage
kubectl top pods -n vigil

# Profile application
curl http://<pod-ip>:9090/debug/pprof/profile?seconds=30

# Check CPU limit
kubectl describe pod <pod-name> -n vigil | grep -A 3 "Limits:"
```

**Solutions:**

| Issue | Solution |
|-------|----------|
| High load | Scale horizontally |
| Inefficient code | Profile and optimize |
| Insufficient limit | Increase CPU limit |

---

### 8. Disk Space Issues

**Symptoms:**
- "No space left on device"
- Write errors
- PVC full alerts

**Diagnosis:**

```bash
# Check PVC usage
kubectl exec -it <pod-name> -n vigil -- df -h

# Check PV usage
kubectl get pv

# Check node disk
kubectl describe node <node-name> | grep -A 5 "Allocated resources:"
```

**Solutions:**

| Issue | Solution |
|-------|----------|
| PVC full | Expand PVC or clean up data |
| Node disk full | Clean up unused resources |
| Log files | Rotate or delete old logs |

---

### 9. Network Issues

**Symptoms:**
- Connection timeouts
- DNS resolution failures
- Intermittent connectivity

**Diagnosis:**

```bash
# Test DNS
kubectl exec -it <pod-name> -n vigil -- nslookup kubernetes

# Test connectivity
kubectl exec -it <pod-name> -n vigil -- ping -c 3 <target>

# Check network policy
kubectl get networkpolicy -n vigil
kubectl describe networkpolicy -n vigil
```

**Solutions:**

| Issue | Solution |
|-------|----------|
| DNS failure | Check CoreDNS logs and config |
| Network policy | Verify policy allows traffic |
| DNS timeout | Check CoreDNS performance |

---

### 10. Certificate/TLS Issues

**Symptoms:**
- Certificate expired
- TLS handshake failures
- Unknown CA errors

**Diagnosis:**

```bash
# Check certificate
kubectl get certificate -n vigil
kubectl describe certificate <cert-name> -n vigil

# Check TLS secret
kubectl get secret <tls-secret> -n vigil -o yaml

# Test TLS
openssl s_client -connect <host>:443 -showcerts
```

**Solutions:**

| Issue | Solution |
|-------|----------|
| Expired certificate | Rotate certificate |
| Unknown CA | Add CA to trust store |
| Invalid certificate | Re-issue certificate |

---

## Diagnostic Commands Quick Reference

```bash
# Cluster health
kubectl get nodes
kubectl get pods -A | grep -v Running

# Resource usage
kubectl top nodes
kubectl top pods -n vigil

# Logs
kubectl logs -f deployment/<service> -n vigil --all-containers

# Events
kubectl get events -n vigil --sort-by='.lastTimestamp'

# Network test
kubectl run nettest --rm -it --image=busybox -- sh

# Port forward
kubectl port-forward svc/<service> <local>:<remote> -n vigil
```

---

**Last Updated:** 2026-04-14
**Version:** 1.0
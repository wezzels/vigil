# VIGIL Operator Runbook

## Overview

This runbook provides operational procedures for VIGIL system operators.

---

## 1. Deployment Procedures

### 1.1 Initial Deployment

```bash
# Create namespace
kubectl apply -f k8s/vigil/namespace.yaml

# Deploy infrastructure
kubectl apply -f k8s/vigil/networkpolicy.yaml
kubectl apply -f k8s/vigil/istio.yaml

# Deploy services
kubectl apply -f k8s/vigil/opir-ingest.yaml
kubectl apply -f k8s/vigil/missile-warning.yaml
kubectl apply -f k8s/vigil/sensor-fusion.yaml
kubectl apply -f k8s/vigil/lvc-coordinator.yaml

# Configure autoscaling
kubectl apply -f k8s/vigil/hpa.yaml
```

### 1.2 Rolling Update

```bash
# Update image
kubectl set image deployment/opir-ingest \
  opir-ingest=vigil/opir-ingest:v1.2.0 \
  -n vigil

# Monitor rollout
kubectl rollout status deployment/opir-ingest -n vigil

# Rollback if needed
kubectl rollout undo deployment/opir-ingest -n vigil
```

### 1.3 Scale Services

```bash
# Manual scale
kubectl scale deployment/opir-ingest --replicas=5 -n vigil

# Update HPA limits
kubectl patch hpa opir-ingest-hpa -n vigil \
  --type=merge -p '{"spec":{"maxReplicas":20}}'
```

---

## 2. Monitoring Procedures

### 2.1 Check System Health

```bash
# Check all pods
kubectl get pods -n vigil

# Check services
kubectl get svc -n vigil

# Check endpoints
kubectl get endpoints -n vigil

# Check health endpoints
curl http://opir-ingest:8080/healthz/liveness
curl http://opir-ingest:8080/healthz/readiness
```

### 2.2 View Logs

```bash
# Pod logs
kubectl logs -f deployment/opir-ingest -n vigil

# All pods with label
kubectl logs -l app=opir-ingest -n vigil

# Previous container logs
kubectl logs deployment/opir-ingest -n vigil --previous
```

### 2.3 Monitor Metrics

```bash
# Port forward to Prometheus
kubectl port-forward svc/prometheus 9090:9090 -n monitoring

# Port forward to Grafana
kubectl port-forward svc/grafana 3000:80 -n monitoring
```

**Key Metrics to Monitor:**

| Metric | Alert Threshold |
|--------|-----------------|
| `vigil_tracks_active` | > 10000 |
| `vigil_alerts_pending` | > 100 |
| `vigil_latency_p99` | > 100ms |
| `vigil_errors_total` | > 10/min |

### 2.4 Check Kafka

```bash
# List topics
kafka-topics --list --bootstrap-server kafka:9092

# Consumer lag
kafka-consumer-groups --describe \
  --group vigil-opir \
  --bootstrap-server kafka:9092

# Topic details
kafka-topics --describe --topic opir-detections \
  --bootstrap-server kafka:9092
```

---

## 3. Incident Procedures

### 3.1 Service Down

```bash
# Check pod status
kubectl describe pod <pod-name> -n vigil

# Check events
kubectl get events -n vigil --sort-by='.lastTimestamp'

# Check resource usage
kubectl top pods -n vigil

# Restart service
kubectl rollout restart deployment/<service> -n vigil
```

### 3.2 High Error Rate

```bash
# Check logs for errors
kubectl logs deployment/<service> -n vigil | grep -i error

# Check error metrics
curl http://<service>:9090/metrics | grep vigil_errors

# Check dependencies
kubectl exec -it deployment/<service> -n vigil -- \
  curl http://kafka:9092/brokers
kubectl exec -it deployment/<service> -n vigil -- \
  curl http://postgres:5432/health
```

### 3.3 Performance Degradation

```bash
# Check resource limits
kubectl describe pod <pod-name> -n vigil | grep -A 5 Limits

# Check HPA status
kubectl get hpa -n vigil

# Check network latency
kubectl exec -it deployment/<service> -n vigil -- \
  curl -w "@curl-format.txt" http://postgres:5432
```

### 3.4 Database Issues

```bash
# Check PostgreSQL status
kubectl exec -it postgres-0 -n vigil -- \
  psql -U vigil -c "SELECT * FROM pg_stat_activity;"

# Check connections
kubectl exec -it postgres-0 -n vigil -- \
  psql -U vigil -c "SELECT count(*) FROM pg_stat_activity;"

# Check replication
kubectl exec -it postgres-0 -n vigil -- \
  psql -U vigil -c "SELECT * FROM pg_stat_replication;"

# Check Patroni
patronictl list
```

### 3.5 Kafka Issues

```bash
# Check broker status
kafka-broker-api-versions --bootstrap-server kafka:9092

# Check under-replicated partitions
kafka-topics --describe --under-replicated-partitions \
  --bootstrap-server kafka:9092

# Check consumer groups
kafka-consumer-groups --list --bootstrap-server kafka:9092

# Reset consumer offset (use with caution!)
kafka-consumer-groups --reset-offsets \
  --group vigil-opir \
  --topic opir-detections \
  --to-latest \
  --execute \
  --bootstrap-server kafka:9092
```

---

## 4. Maintenance Procedures

### 4.1 Database Backup

```bash
# Manual backup
velero backup create vigil-backup-$(date +%Y%m%d) \
  --include-namespaces vigil

# Verify backup
velero backup describe vigil-backup-$(date +%Y%m%d)

# List backups
velero backup get
```

### 4.2 Database Restore

```bash
# Restore from backup
velero restore create --from-backup vigil-backup-20260414

# Verify restore
velero restore describe <restore-name>
```

### 4.3 Certificate Rotation

```bash
# Check certificate expiration
kubectl get certificates -n vigil

# Rotate certificate
kubectl annotate certificate vigil-tls \
  cert-manager.io/issue-temporary-certificate=true \
  -n vigil

# Verify new certificate
kubectl get certificate vigil-tls -n vigil -o yaml
```

### 4.4 Configuration Update

```bash
# Update ConfigMap
kubectl edit configmap vigil-config -n vigil

# Restart pods to pick up changes
kubectl rollout restart deployment -n vigil
```

---

## 5. Troubleshooting Commands

### General

```bash
# Get all resources
kubectl get all -n vigil

# Describe resource
kubectl describe <resource> <name> -n vigil

# Execute command in pod
kubectl exec -it <pod-name> -n vigil -- /bin/sh

# Port forward
kubectl port-forward svc/<service> <local-port>:<remote-port> -n vigil
```

### Network

```bash
# Test connectivity
kubectl exec -it <pod-name> -n vigil -- curl http://kafka:9092

# DNS lookup
kubectl exec -it <pod-name> -n vigil -- nslookup kafka

# Check network policy
kubectl get networkpolicy -n vigil
kubectl describe networkpolicy <policy-name> -n vigil
```

### Storage

```bash
# List PVCs
kubectl get pvc -n vigil

# Check volume
kubectl describe pvc <pvc-name> -n vigil

# List volumes
kubectl get pv | grep vigil
```

---

## 6. Escalation

### Escalation Path

| Level | Contact | Response Time |
|-------|---------|---------------|
| L1 | On-Call Engineer | 15 min |
| L2 | Platform Lead | 30 min |
| L3 | Engineering Manager | 1 hour |

### Incident Classification

| Severity | Description | Response |
|----------|-------------|----------|
| P1 | System down | Immediate |
| P2 | Degraded service | 30 min |
| P3 | Minor issue | 4 hours |

---

## 7. Contacts

| Role | Name | Phone | Email |
|------|------|-------|-------|
| On-Call | TBD | TBD | oncall@vigil.local |
| Platform Lead | TBD | TBD | platform@vigil.local |
| DBA | TBD | TBD | dba@vigil.local |
| Security | TBD | TBD | security@vigil.local |

---

**Last Updated:** 2026-04-14
**Version:** 1.0
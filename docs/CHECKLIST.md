# VIGIL Deployment Checklist

## Pre-Deployment Checklist

### Infrastructure

- [ ] Kubernetes cluster is running and healthy
- [ ] All nodes are Ready
- [ ] Sufficient CPU resources available (minimum 32 cores total)
- [ ] Sufficient memory available (minimum 128 GB total)
- [ ] Storage provisioner is configured
- [ ] Network policies are ready

### Dependencies

- [ ] PostgreSQL is deployed and accessible
- [ ] TimescaleDB extension is installed
- [ ] Redis is deployed and accessible
- [ ] Kafka cluster is running (3 brokers minimum)
- [ ] Prometheus is deployed
- [ ] Grafana is deployed

### Security

- [ ] TLS certificates are generated
- [ ] Secrets are created in namespace
- [ ] Service accounts are configured
- [ ] RBAC is configured
- [ ] Network policies are ready

### Configuration

- [ ] ConfigMaps are populated
- [ ] Environment variables are set
- [ ] Resource limits are defined
- [ ] Health checks are configured

---

## Deployment Checklist

### Step 1: Namespace Setup

```bash
# Create namespace
kubectl apply -f k8s/vigil/namespace.yaml

# Verify
kubectl get namespace vigil
kubectl get serviceaccount -n vigil
kubectl get clusterrole vigil-role
kubectl get clusterrolebinding vigil-binding
```

- [ ] Namespace created
- [ ] ServiceAccount created
- [ ] ClusterRole created
- [ ] ClusterRoleBinding created

### Step 2: Secrets

```bash
# Verify secrets exist
kubectl get secrets -n vigil

# Check secret content (redacted)
kubectl describe secret vigil-secrets -n vigil
```

- [ ] vigil-secrets exists
- [ ] POSTGRES_PASSWORD set
- [ ] KAFKA_SASL_PASSWORD set (if applicable)
- [ ] TLS certificate secret exists

### Step 3: ConfigMaps

```bash
# Verify ConfigMaps
kubectl get configmaps -n vigil
kubectl describe configmap vigil-config -n vigil
```

- [ ] vigil-config exists
- [ ] POSTGRES_HOST set
- [ ] KAFKA_BROKERS set
- [ ] REDIS_HOST set

### Step 4: Network Policies

```bash
# Apply network policies
kubectl apply -f k8s/vigil/networkpolicy.yaml

# Verify
kubectl get networkpolicy -n vigil
```

- [ ] Default deny policies applied
- [ ] Service-specific policies applied
- [ ] DNS policy applied

### Step 5: Istio (Optional)

```bash
# Apply Istio config
kubectl apply -f k8s/vigil/istio.yaml

# Verify
kubectl get peerauthentication -n vigil
kubectl get authorizationpolicy -n vigil
```

- [ ] PeerAuthentication configured
- [ ] AuthorizationPolicy configured
- [ ] DestinationRules configured
- [ ] VirtualServices configured

### Step 6: Deploy Services

```bash
# Deploy OPIR Ingest
kubectl apply -f k8s/vigil/opir-ingest.yaml
kubectl rollout status deployment/opir-ingest -n vigil

# Deploy Missile Warning
kubectl apply -f k8s/vigil/missile-warning.yaml
kubectl rollout status deployment/missile-warning -n vigil

# Deploy Sensor Fusion
kubectl apply -f k8s/vigil/sensor-fusion.yaml
kubectl rollout status deployment/sensor-fusion -n vigil

# Deploy LVC Coordinator
kubectl apply -f k8s/vigil/lvc-coordinator.yaml
kubectl rollout status deployment/lvc-coordinator -n vigil
```

- [ ] OPIR Ingest deployed and running
- [ ] Missile Warning deployed and running
- [ ] Sensor Fusion deployed and running
- [ ] LVC Coordinator deployed and running

### Step 7: Horizontal Pod Autoscalers

```bash
# Apply HPAs
kubectl apply -f k8s/vigil/hpa.yaml

# Verify
kubectl get hpa -n vigil
```

- [ ] OPIR Ingest HPA configured
- [ ] Missile Warning HPA configured
- [ ] Sensor Fusion HPA configured
- [ ] Pod Disruption Budgets configured

### Step 8: Database Migrations

```bash
# Run migrations
kubectl exec -it postgres-0 -n vigil -- \
  psql -U vigil -f /migrations/001_tracks.up.sql

# Verify tables
kubectl exec -it postgres-0 -n vigil -- \
  psql -U vigil -c "\dt"
```

- [ ] Tracks table created
- [ ] Alerts table created
- [ ] Events table created
- [ ] TimescaleDB hypertables created

---

## Post-Deployment Checklist

### Health Checks

```bash
# Check all pods are Running
kubectl get pods -n vigil

# Check services have endpoints
kubectl get endpoints -n vigil

# Test health endpoints
kubectl run curl --rm -it --image=curlimages/curl -- \
  curl http://opir-ingest.vigil:8080/healthz/liveness
kubectl run curl --rm -it --image=curlimages/curl -- \
  curl http://missile-warning.vigil:8080/healthz/readiness
```

- [ ] All pods in Running state
- [ ] All services have endpoints
- [ ] Liveness probes passing
- [ ] Readiness probes passing
- [ ] Startup probes passing

### Functionality Tests

```bash
# Test API endpoint
kubectl run curl --rm -it --image=curlimages/curl -- \
  curl http://sensor-fusion.vigil:8080/api/tracks

# Test alert endpoint
kubectl run curl --rm -it --image=curlimages/curl -- \
  curl http://missile-warning.vigil:8080/api/alerts

# Test metrics endpoint
kubectl run curl --rm -it --image=curlimages/curl -- \
  curl http://opir-ingest.vigil:9090/metrics
```

- [ ] API endpoints respond
- [ ] Tracks API works
- [ ] Alerts API works
- [ ] Metrics endpoint works

### Monitoring

```bash
# Check Prometheus targets
kubectl port-forward svc/prometheus 9090:9090 -n monitoring
# Open http://localhost:9090/targets

# Check Grafana dashboards
kubectl port-forward svc/grafana 3000:80 -n monitoring
# Open http://localhost:3000
```

- [ ] Prometheus scraping targets
- [ ] Grafana dashboards loaded
- [ ] Alerts configured
- [ ] Service dashboards working

### Logging

```bash
# Check logs are flowing
kubectl logs -l app=opir-ingest -n vigil --tail=50
```

- [ ] Application logs visible
- [ ] Log format is correct
- [ ] No error logs

### Backup Verification

```bash
# Check Velero backups
velero backup get

# Verify backup schedule
velero schedule get
```

- [ ] Backup schedule configured
- [ ] Recent backup completed
- [ ] Backup is valid

---

## Rollback Checklist

If deployment fails:

```bash
# Rollback deployments
kubectl rollout undo deployment/opir-ingest -n vigil
kubectl rollout undo deployment/missile-warning -n vigil
kubectl rollout undo deployment/sensor-fusion -n vigil
kubectl rollout undo deployment/lvc-coordinator -n vigil

# Verify rollback
kubectl rollout status deployment/opir-ingest -n vigil
```

- [ ] Rollback completed
- [ ] Services restored
- [ ] Health checks passing

---

## Sign-off

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Deployer | | | |
| Operator | | | |
| QA | | | |
| Approver | | | |

---

**Version:** 1.0
**Last Updated:** 2026-04-14
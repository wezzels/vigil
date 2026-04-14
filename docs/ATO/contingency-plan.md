# VIGIL Contingency Plan

## Overview

This document describes the contingency planning for VIGIL system failures and recovery procedures.

---

## 1. Contingency Levels

### Level 1: Minor Incident
- Single service failure
- Performance degradation
- Non-critical component failure
- **Response**: Automated recovery via Kubernetes
- **RTO**: < 5 minutes
- **RPO**: < 1 minute

### Level 2: Major Incident
- Multiple service failures
- Database failover required
- Significant performance impact
- **Response**: Manual intervention, automated recovery
- **RTO**: < 30 minutes
- **RPO**: < 5 minutes

### Level 3: Disaster
- Data center failure
- Complete system outage
- Data loss scenario
- **Response**: DR site activation
- **RTO**: < 4 hours
- **RPO**: < 1 hour

---

## 2. Recovery Procedures

### 2.1 Service Recovery

#### Single Pod Failure

```bash
# Kubernetes automatically restarts failed pods
# Monitor recovery:
kubectl get pods -n vigil -w

# If pod doesn't recover:
kubectl describe pod <pod-name> -n vigil
kubectl logs <pod-name> -n vigil --previous
kubectl delete pod <pod-name> -n vigil
```

#### Service Degradation

```bash
# Check service health:
kubectl get svc -n vigil
kubectl get endpoints -n vigil

# Restart service:
kubectl rollout restart deployment/<service> -n vigil

# Scale up if needed:
kubectl scale deployment/<service> -n vigil --replicas=5
```

### 2.2 Database Recovery

#### PostgreSQL Primary Failover

```bash
# Patroni automatically handles failover
# Check cluster status:
patronictl list

# Manual switchover if needed:
patronictl switchover --master <old-master> --candidate <new-master>

# Verify replication:
psql -c "SELECT * FROM pg_stat_replication;"
```

#### Database Backup Restore

```bash
# List available backups:
velero backup get

# Restore from backup:
velero restore create --from-backup <backup-name>

# Verify restore:
velero restore describe <restore-name>
```

### 2.3 Kafka Recovery

#### Broker Failure

```bash
# Kafka automatically replicates with replication factor 3
# Check under-replicated partitions:
kafka-topics --describe --under-replicated-partitions \
  --bootstrap-server kafka:9092

# Reassign partitions if needed:
kafka-reassign-partitions --reassignment-json-file reassign.json \
  --bootstrap-server kafka:9092
```

### 2.4 Redis Recovery

#### Redis Failover

```bash
# Check Redis status:
redis-cli info replication

# Manual failover if needed:
redis-cli cluster failover
```

---

## 3. Disaster Recovery

### 3.1 DR Site Activation

#### Pre-requisites

1. DR site infrastructure operational
2. Network connectivity established
3. Backup data available
4. DNS failover configured

#### Activation Procedure

```bash
# 1. Verify DR site readiness
ansible-playbook -i dr-inventory.yaml check-dr-site.yaml

# 2. Restore from backup
velero restore create --from-backup daily-backup-<date>

# 3. Verify data integrity
kubectl exec -n vigil postgres -- psql -c "SELECT COUNT(*) FROM tracks;"

# 4. Start services
kubectl apply -f k8s/vigil/ -n vigil

# 5. Update DNS
# Update DNS A records to point to DR site

# 6. Verify functionality
curl https://vigil.local/healthz
```

### 3.2 Failback Procedure

```bash
# 1. Synchronize data from DR to primary
pg_dump -h dr-postgres vigil | psql -h primary-postgres vigil

# 2. Stop DR services
kubectl scale deployment --all -n vigil --replicas=0

# 3. Start primary services
kubectl apply -f k8s/vigil/ -n vigil

# 4. Update DNS
# Update DNS A records to point to primary site

# 5. Verify functionality
curl https://vigil.local/healthz
```

---

## 4. Communication Plan

### 4.1 Incident Notification

| Level | Notification | Channels |
|-------|--------------|----------|
| 1 | Operations team | Slack, Email |
| 2 | Management + Ops | Slack, Email, Phone |
| 3 | All stakeholders | Slack, Email, Phone, SMS |

### 4.2 Escalation Matrix

| Time | Level 1 Contact | Level 2 Contact | Level 3 Contact |
|------|-----------------|-----------------|-----------------|
| Business Hours | On-Call Engineer | Shift Lead | Operations Manager |
| After Hours | On-Call Engineer | On-Call Lead | On-Call Manager |
| Weekend | Weekend Engineer | Weekend Lead | Weekend Manager |

---

## 5. Testing Schedule

### 5.1 Recovery Testing

| Test | Frequency | Scope | Success Criteria |
|------|-----------|-------|-------------------|
| Pod recovery | Weekly | Single pod failure | Auto-recovery < 5 min |
| Service restart | Monthly | Service deployment | Recovery < 10 min |
| Database failover | Monthly | Patroni failover | Failover < 30 sec |
| Backup restore | Quarterly | Full restore | Restore < 4 hours |
| DR failover | Annually | Full DR activation | Activation < 4 hours |

### 5.2 Test Documentation

#### Pre-Test Checklist

- [ ] Test plan approved
- [ ] Backup verified
- [ ] Monitoring confirmed
- [ ] Communication plan ready
- [ ] Rollback plan documented

#### Post-Test Checklist

- [ ] Test results documented
- [ ] Lessons learned captured
- [ ] Runbook updated
- [ ] Metrics recorded

---

## 6. Backup and Recovery

### 6.1 Backup Schedule

| Backup Type | Frequency | Retention | Location |
|-------------|-----------|-----------|----------|
| PostgreSQL hourly | Hourly | 24 hours | MinIO |
| PostgreSQL daily | Daily | 30 days | MinIO + Offsite |
| Full cluster | Weekly | 4 weeks | MinIO + Offsite |
| Configuration | On change | 90 days | Git |

### 6.2 Recovery Time Objectives

| Component | RTO | RPO | Method |
|-----------|-----|-----|--------|
| Application services | 5 min | 0 | Kubernetes restart |
| PostgreSQL | 5 min | < 1 min | Patroni failover |
| Redis | 5 min | 0 | Cluster failover |
| Kafka | 10 min | 0 | Replication |
| Full system | 4 hours | 1 hour | DR site |

---

## 7. Appendices

### 7.1 Contact Information

| Role | Name | Phone | Email |
|------|------|-------|-------|
| System Owner | TBD | TBD | owner@vigil.local |
| ISSO | TBD | TBD | isso@vigil.local |
| Platform Lead | TBD | TBD | platform@vigil.local |
| DBA | TBD | TBD | dba@vigil.local |

### 7.2 Service Dependencies

```
OPIR Ingest
  └── Kafka
  └── Redis
  └── PostgreSQL

Missile Warning
  └── Kafka
  └── PostgreSQL
  └── TimescaleDB

Sensor Fusion
  └── Kafka
  └── PostgreSQL
  └── TimescaleDB
  └── Redis

LVC Coordinator
  └── Kafka
  └── Redis
```

---

**Document Version**: 1.0
**Date**: 2026-04-14
**Review Date**: 2026-10-14
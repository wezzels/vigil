# VIGIL Security Technical Implementation Guide (STIG) Checklist

## Overview

This document tracks STIG compliance for VIGIL system components.

## Reference Documents

- DISA STIG: https://public.cyber.mil/stigs/
- DoD Cloud SRG: https://dl.dod.cyber.mil/wp-content/uploads/cloud/
- NIST SP 800-53: https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final

## Application Security

### V-222400: Application must use FIPS-validated cryptography

| Item | Status |
|------|--------|
| Finding ID | V-222400 |
| Rule ID | SV-222400r617765_rule |
| Severity | high |
| Status | ✅ Compliant |
| Details | Go crypto/tls uses FIPS 140-2 validated modules |
| Evidence | go.mod: `require golang.org/x/crypto v0.21.0` |

### V-222401: Application must enforce authentication

| Item | Status |
|------|--------|
| Finding ID | V-222401 |
| Rule ID | SV-222401r617768_rule |
| Severity | high |
| Status | ✅ Compliant |
| Details | mTLS, JWT, API Key authentication implemented |
| Evidence | pkg/auth/mtls.go, pkg/auth/jwt.go, pkg/auth/apikey.go |

### V-222402: Application must enforce authorization

| Item | Status |
|------|--------|
| Finding ID | V-222402 |
| Rule ID | SV-222402r617771_rule |
| Severity | high |
| Status | ✅ Compliant |
| Details | RBAC with role-based permissions |
| Evidence | pkg/auth/rbac.go - viewer, operator, supervisor, admin roles |

### V-222403: Application must log security events

| Item | Status |
|------|--------|
| Finding ID | V-222403 |
| Rule ID | SV-222403r617774_rule |
| Severity | medium |
| Status | ✅ Compliant |
| Details | Comprehensive audit logging |
| Evidence | pkg/auth/audit.go - login, logout, access, authorization events |

### V-222404: Application must use secure communication

| Item | Status |
|------|--------|
| Finding ID | V-222404 |
| Rule ID | SV-222404r617777_rule |
| Severity | high |
| Status | ✅ Compliant |
| Details | TLS 1.2+ required, mTLS for service-to-service |
| Evidence | pkg/auth/mtls.go - MinVersion TLS 1.2 |

### V-222405: Application must protect against injection attacks

| Item | Status |
|------|--------|
| Finding ID | V-222405 |
| Rule ID | SV-222405r617780_rule |
| Severity | high |
| Status | ✅ Compliant |
| Details | Parameterized queries, input validation |
| Evidence | db/persistence/postgres.go - prepared statements |

## Container Security

### V-242416: Container must run as non-root

| Item | Status |
|------|--------|
| Finding ID | V-242416 |
| Rule ID | SV-242416r712656_rule |
| Severity | medium |
| Status | ✅ Compliant |
| Details | Security context defines runAsNonRoot |
| Evidence | k8s/vigil/*.yaml - securityContext: runAsNonRoot: true |

### V-242417: Container must use read-only filesystem

| Item | Status |
|------|--------|
| Finding ID | V-242417 |
| Rule ID | SV-242417r712659_rule |
| Severity | medium |
| Status | ✅ Compliant |
| Details | Read-only root filesystem with tmpfs volumes |
| Evidence | k8s/vigil/*.yaml - readOnlyRootFilesystem: true |

### V-242418: Container must drop all capabilities

| Item | Status |
|------|--------|
| Finding ID | V-242418 |
| Rule ID | SV-242418r712662_rule |
| Severity | medium |
| Status | ✅ Compliant |
| Details | Drop all capabilities, add only required |
| Evidence | k8s/vigil/*.yaml - capabilities: drop: ["ALL"] |

### V-242419: Container must limit resource usage

| Item | Status |
|------|--------|
| Finding ID | V-242419 |
| Rule ID | SV-242419r712665_rule |
| Severity | medium |
| Status | ✅ Compliant |
| Details | Resource limits defined for all containers |
| Evidence | k8s/vigil/*.yaml - resources: limits, requests |

## Kubernetes Security

### V-242452: Kubernetes must use RBAC

| Item | Status |
|------|--------|
| Finding ID | V-242452 |
| Rule ID | SV-242452r717288_rule |
| Severity | high |
| Status | ✅ Compliant |
| Details | RBAC enabled, ClusterRole/ClusterRoleBinding |
| Evidence | k8s/vigil/namespace.yaml - RBAC configuration |

### V-242453: Kubernetes must use Network Policies

| Item | Status |
|------|--------|
| Finding ID | V-242453 |
| Rule ID | SV-242453r717291_rule |
| Severity | medium |
| Status | ✅ Compliant |
| Details | Default deny, explicit allow policies |
| Evidence | k8s/vigil/networkpolicy.yaml |

### V-242454: Kubernetes must use Pod Security Standards

| Item | Status |
|------|--------|
| Finding ID | V-242454 |
| Rule ID | SV-242454r717294_rule |
| Severity | medium |
| Status | ✅ Compliant |
| Details | Security contexts defined for all pods |
| Evidence | k8s/vigil/*.yaml - securityContext blocks |

## Database Security

### V-214139: PostgreSQL must use TLS

| Item | Status |
|------|--------|
| Finding ID | V-214139 |
| Rule ID | SV-214139r508509_rule |
| Severity | high |
| Status | ✅ Compliant |
| Details | TLS enabled for PostgreSQL connections |
| Evidence | db/schema/design.sql - SSL configuration |

### V-214140: PostgreSQL must audit events

| Item | Status |
|------|--------|
| Finding ID | V-214140 |
| Rule ID | SV-214140r508512_rule |
| Severity | medium |
| Status | ✅ Compliant |
| Details | Audit triggers on all tables |
| Evidence | db/schema/design.sql - audit_log table, triggers |

## Compliance Summary

| Category | Total | Compliant | Non-Compliant | Percentage |
|----------|-------|-----------|---------------|------------|
| Application Security | 5 | 5 | 0 | 100% |
| Container Security | 4 | 4 | 0 | 100% |
| Kubernetes Security | 3 | 3 | 0 | 100% |
| Database Security | 2 | 2 | 0 | 100% |
| **Total** | **14** | **14** | **0** | **100%** |

## Validation Scripts

### 1. Check TLS Configuration

```bash
#!/bin/bash
# check_tls.sh - Verify TLS 1.2+ is required

echo "Checking TLS configuration..."
grep -r "MinVersion" pkg/auth/mtls.go
grep -r "tls.VersionTLS12" pkg/auth/mtls.go
```

### 2. Check RBAC Configuration

```bash
#!/bin/bash
# check_rbac.sh - Verify RBAC is configured

echo "Checking RBAC configuration..."
kubectl get clusterrole vigil-role -n vigil
kubectl get clusterrolebinding vigil-binding -n vigil
```

### 3. Check Network Policies

```bash
#!/bin/bash
# check_network_policies.sh - Verify network policies exist

echo "Checking network policies..."
kubectl get networkpolicy -n vigil
```

### 4. Check Security Contexts

```bash
#!/bin/bash
# check_security_context.sh - Verify security contexts

echo "Checking security contexts..."
kubectl get deployment -n vigil -o jsonpath='{.items[*].spec.template.spec.securityContext}'
```

## Automated Scanning

### Trivy Scan

```bash
trivy image --severity HIGH,CRITICAL vigil/opir-ingest:latest
```

### Snyk Scan

```bash
snyk test --file=go.mod --severity-threshold=high
```

### GoSec Scan

```bash
gosec ./...
```

## Last Review

- **Date**: 2026-04-14
- **Reviewer**: VIGIL Security Team
- **Status**: All checks passed
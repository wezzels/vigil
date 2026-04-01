#!/bin/bash
# forge-status.sh — CICERONE FORGE status command
# Usage: cicerone forge status [--federation NAME]

set -euo pipefail

FEDERATION="${1:-TROOPER-FORGE}"
NAMESPACE="${NAMESPACE:-forge}"
KUBECTL="${KUBECTL:-kubectl}"

echo "=== TROOPER-FORGE Status ==="
echo "Federation: $FEDERATION"
echo "Namespace: $NAMESPACE"
echo ""

echo "--- K8s Services ---"
$KUBECTL get svc -n $NAMESPACE 2>/dev/null || echo "Namespace not found or kubectl unavailable"

echo ""
echo "--- K8s Pods ---"
$KUBECTL get pods -n $NAMESPACE 2>/dev/null || echo "Cannot reach cluster"

echo ""
echo "--- Kafka Topics ---"
kubectl exec -n gms deployment/kafka -- kafka-topics.sh --bootstrap-server localhost:9092 --list 2>/dev/null || echo "Kafka not accessible"

echo ""
echo "--- DIS Federation ---"
ss -tlnp | grep 3000 || echo "DIS port 3000 not listening"

echo ""
echo "--- etcd ---"
kubectl exec -n gms statefulset/etcd -- etcdctl --endpoints localhost:2379 member list 2>/dev/null || echo "etcd not accessible"

echo ""
echo "--- Redis ---"
kubectl exec -n gms deployment/redis -- redis-cli ping 2>/dev/null || echo "Redis not accessible"

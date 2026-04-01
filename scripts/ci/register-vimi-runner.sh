#!/bin/bash
# VIMI GitLab Runner Registration Script
# ======================================
# Run this AFTER getting the runner token from GitLab UI:
#   https://idm.wezzel.com/crab-meat-repos/trooper-vimi/-/settings/ci_cd
#   → Set up a specific runner → Register a runner → copy the token
#
# Usage: sudo bash register-vimi-runner.sh <RUNNER_TOKEN>
#
set -e

TOKEN="${1:-}"
if [[ -z "$TOKEN" ]]; then
    echo "Usage: sudo bash $0 <RUNNER_TOKEN>"
    echo ""
    echo "Get token at: https://idm.wezzel.com/crab-meat-repos/trooper-vimi/-/settings/ci_cd"
    echo "  → Expand 'Specific runners'"
    echo "  → Click 'Register a runner'"
    echo "  → Copy the token (looks like: GR134894...)"
    exit 1
fi

echo "Registering VIMI GitLab runner..."
gitlab-runner register \
  --non-interactive \
  --url "https://idm.wezzel.com" \
  --registration-token "$TOKEN" \
  --description "vimi-k8s-runner" \
  --executor "docker" \
  --docker-image "docker:24-dind" \
  --docker-privileged \
  --docker-volumes "/cache" \
  --tag-list "vimi,kubernetes,docker" \
  --locked-to-project "false" \
  --run-untagged "false" \
  --access-level "not_protected"

echo "Starting runner..."
gitlab-runner start
gitlab-runner verify

echo ""
echo "=== Runner registered successfully ==="
gitlab-runner list

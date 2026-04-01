#!/bin/bash
# federation-join.sh — Join a DIS/HLA federation
# Usage: cicerone federation join --type DIS --name NAME --multicast ADDR --port PORT

set -euo pipefail

TYPE="${TYPE:-DIS}"
NAME="${NAME:-TROOPER-VIMI}"
MULTICAST="${MULTICAST:-239.255.0.1}"
PORT="${PORT:-3000}"
EXERCISE="${EXERCISE:-1}"

echo "Joining $TYPE federation: $NAME"
echo "  Multicast: $MULTICAST:$PORT"
echo "  Exercise: $EXERCISE"
echo ""

# Validate multicast range (DIS uses 239.255.0.0 - 239.255.255.255)
IFS='.' read -r a b c d <<< "$MULTICAST"
if [ "$a" -ne 239 ] || [ "$b" -ne 255 ]; then
    echo "ERROR: DIS multicast must be 239.255.x.x"
    exit 1
fi

# For DIS: set up UDP multicast subscription
# This is handled by the lvc-coordinator service
echo "Note: Use lvc-coordinator service to manage federation membership"
echo "Joining federation via Kafka event bus..."

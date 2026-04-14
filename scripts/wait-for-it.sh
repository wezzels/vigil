#!/bin/bash
# wait-for-it.sh - Wait for service to be ready
# Usage: wait-for-it.sh host:port -- command args

TIMEOUT=15
QUIET=0

echoerr() {
  if [ $QUIET -eq 0 ]; then echo "$@" 1>&2; fi
}

usage() {
  echo "Usage: $0 host:port [-t timeout] [-- command args]"
  echo "  -t TIMEOUT  Timeout in seconds (default: 15)"
  echo "  -q          Quiet mode"
  echo "  -- COMMAND  Command to execute after service is ready"
  exit 1
}

wait_for() {
  for i in `seq $TIMEOUT`; do
    nc -z $HOST $PORT >/dev/null 2>&1
    result=$?
    if [ $result -eq 0 ]; then
      if [ $# -gt 0 ]; then
        exec "$@"
      fi
      exit 0
    fi
    sleep 1
  done
  echoerr "Timeout after $TIMEOUT seconds waiting for $HOST:$PORT"
  exit 1
}

while [ $# -gt 0 ]; do
  case "$1" in
    *:* )
    HOST=${1%:*}
    PORT=${1#*:}
    shift 1
    ;;
    -t)
    TIMEOUT="$2"
    if [ -z "$TIMEOUT" ]; then break; fi
    shift 2
    ;;
    -q)
    QUIET=1
    shift 1
    ;;
    --)
    shift
    break
    ;;
    *)
    usage
    ;;
  esac
done

if [ -z "$HOST" ] || [ -z "$PORT" ]; then
  usage
fi

wait_for "$@"
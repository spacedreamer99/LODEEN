#!/usr/bin/env bash
# LODEEN dev: rebuild + run server + two clients.
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT="$(pwd)"
BIN="$ROOT/bin"
LOG="$ROOT/tmp"
mkdir -p "$LOG"

echo "==> building"
make build

cleanup() {
    echo
    echo "==> stopping"
    pkill -P $$ 2>/dev/null || true
    for pid in "${SERVER_PID:-}" "${CLIENT1_PID:-}" "${CLIENT2_PID:-}"; do
        [[ -n "$pid" ]] && kill "$pid" 2>/dev/null || true
    done
    wait 2>/dev/null || true
}
trap cleanup EXIT

echo "==> starting server"
"$BIN/lodeen-server" >"$LOG/server.log" 2>&1 &
SERVER_PID=$!

for i in $(seq 1 50); do
    curl -sf http://localhost:9091/healthz >/dev/null 2>&1 && break
    sleep 0.1
done

echo "==> starting pilot1"
LODEEN_CLIENT_NICK=pilot1 "$BIN/lodeen-client" >"$LOG/client1.log" 2>&1 &
CLIENT1_PID=$!

sleep 1

echo "==> starting pilot2"
LODEEN_CLIENT_NICK=pilot2 "$BIN/lodeen-client" >"$LOG/client2.log" 2>&1 &
CLIENT2_PID=$!

echo
echo "server:   $LOG/server.log"
echo "client1:  $LOG/client1.log"
echo "client2:  $LOG/client2.log"
echo
echo "Ctrl+C to stop everything"
wait

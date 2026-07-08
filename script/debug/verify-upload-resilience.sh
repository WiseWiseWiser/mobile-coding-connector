#!/usr/bin/env bash
# Verification script for upload-resilience goal. Writes evidence to SCRATCH.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SCRATCH="${SCRATCH:-/var/folders/s_/nd3t_zbx61747w0qdryxh4wm0000gp/T/grok-goal-407a13081939/implementer}"
mkdir -p "$SCRATCH"

cd "$ROOT"

echo "=== doctest vet ===" | tee "$SCRATCH/upload-resilience-doctest.log"
doctest vet ./client/tests/upload-resilience 2>&1 | tee -a "$SCRATCH/upload-resilience-doctest.log"

echo "=== git diff client/tests/upload-resilience ===" | tee -a "$SCRATCH/upload-resilience-doctest.log"
if git diff ./client/tests/upload-resilience | tee -a "$SCRATCH/upload-resilience-doctest.log" | grep -q .; then
  echo "FAIL: sealed tests modified" | tee -a "$SCRATCH/upload-resilience-doctest.log"
  exit 1
fi
echo "(clean)" | tee -a "$SCRATCH/upload-resilience-doctest.log"

echo "=== doctest test -v (with evidence lines) ===" | tee -a "$SCRATCH/upload-resilience-doctest.log"
doctest test -v ./client/tests/upload-resilience/... 2>&1 | tee -a "$SCRATCH/upload-resilience-doctest.log"

echo "=== evidence grep ===" | tee -a "$SCRATCH/upload-resilience-doctest.log"
grep -E 'evidence: InitCount=|TransportAttempts' "$SCRATCH/upload-resilience-doctest.log" | tee -a "$SCRATCH/upload-resilience-doctest.log" || {
  echo "FAIL: missing evidence lines in doctest log" | tee -a "$SCRATCH/upload-resilience-doctest.log"
  exit 1
}

echo "=== go test ./client/... ===" | tee "$SCRATCH/client-go-test.log"
go test ./client/... 2>&1 | tee "$SCRATCH/client-go-test.log"

echo "=== go test ./cmd/agentcli/... ===" | tee "$SCRATCH/agentcli-go-test.log"
go test ./cmd/agentcli/... 2>&1 | tee "$SCRATCH/agentcli-go-test.log"

echo "=== verification complete ==="
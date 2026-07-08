#!/usr/bin/env python3
"""Analyze keep-alive log to confirm startup-timeout kill loop hypothesis."""
import re
import sys
from datetime import datetime
from pathlib import Path

LOG = Path(sys.argv[1] if len(sys.argv) > 1 else "/root/ai-critic-server-keep-alive.log")
FOCUS_PREFIX = sys.argv[2] if len(sys.argv) > 2 else "2026-07-08T14:"

pat = re.compile(r"\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\] (.*)")


def parse_ts(s: str) -> datetime:
    return datetime.strptime(s, "%Y-%m-%dT%H:%M:%S")


def seconds(a: str, b: str) -> float:
    return (parse_ts(b) - parse_ts(a)).total_seconds()


lines = LOG.read_text(errors="replace").splitlines()

# --- Pair: Starting -> Server started -> fail/ready ---
cycles = []
i = 0
while i < len(lines):
    m = pat.match(lines[i])
    if not m or FOCUS_PREFIX not in m.group(1):
        i += 1
        continue
    ts, msg = m.group(1), m.group(2)
    if "Starting ai-critic server on port" not in msg:
        i += 1
        continue

    start_ts = ts
    spawn_ts = fail_ts = ready_ts = None
    pid = None
    waited_ms = None
    refused = dns = tcp_ok = ping_to = 0
    j = i + 1
    while j < len(lines):
        m2 = pat.match(lines[j])
        if not m2:
            j += 1
            continue
        ts2, msg2 = m2.group(1), m2.group(2)
        if (
            "Starting ai-critic server on port" in msg2
            and j > i + 1
        ):
            break
        if msg2.startswith("Server started (PID=") and spawn_ts is None:
            spawn_ts = ts2
            pm = re.search(r"PID=(\d+)", msg2)
            pid = pm.group(1) if pm else "?"
        elif spawn_ts and pid and "connection refused" in msg2:
            refused += 1
        elif spawn_ts and "lookup localhost: i/o timeout" in msg2:
            dns += 1
        elif spawn_ts and "TCP connection successful" in msg2:
            tcp_ok += 1
        elif spawn_ts and "HTTP ping request failed" in msg2 and "deadline exceeded" in msg2:
            ping_to += 1
        elif "failed to become ready within 10s" in msg2:
            fail_ts = ts2
            j += 1
            break
        elif pid and f"phase=server_ready" in msg2 and f"pid={pid}" in msg2:
            ready_ts = ts2
            wm = re.search(r"waited_ms=(\d+)", msg2)
            waited_ms = int(wm.group(1)) if wm else None
            j += 1
            break
        j += 1

    if spawn_ts:
        cycles.append(
            {
                "pid": pid,
                "start": start_ts,
                "spawn": spawn_ts,
                "fail": fail_ts,
                "ready": ready_ts,
                "waited_ms": waited_ms,
                "refused": refused,
                "dns": dns,
                "tcp_ok": tcp_ok,
                "ping_to": ping_to,
                "exec_delay": seconds(start_ts, spawn_ts),
                "wait_window": seconds(spawn_ts, fail_ts) if fail_ts else None,
            }
        )
    i = j if j > i else i + 1

failed = [c for c in cycles if c["fail"]]
ready = [c for c in cycles if c["ready"]]

print(f"Log: {LOG}")
print(f"Focus: {FOCUS_PREFIX}*")
print(f"Cycles with spawn: {len(cycles)} (failed={len(failed)}, ready={len(ready)})")
print()

print("=== GLOBAL COUNTS (whole log) ===")
text = LOG.read_text(errors="replace")
for label, needle in [
    ("server_spawn", "phase=server_spawn"),
    ("server_ready", "phase=server_ready"),
    ("failed_10s", "failed to become ready within 10s"),
    ("killing", "Killing process group"),
    ("port_check_ok", "phase=port_check_ok"),
    ("conn_refused", "connection refused"),
    ("dns_timeout", "lookup localhost: i/o timeout"),
]:
    print(f"  {label}: {text.count(needle)}")
print()

print("=== FAILED CYCLES (first 20) ===")
print("pid | exec_delay | wait_window | refused | dns | tcp_ok | ping_to")
for c in failed[:20]:
    print(
        f"{c['pid']} | {c['exec_delay']:.0f}s | {c['wait_window']:.0f}s | "
        f"{c['refused']} | {c['dns']} | {c['tcp_ok']} | {c['ping_to']}"
    )
if len(failed) > 20:
    print(f"... +{len(failed) - 20} more")

print()
print("=== READY CYCLES ===")
for c in ready:
    print(
        f"pid={c['pid']} exec_delay={c['exec_delay']:.0f}s waited_ms={c['waited_ms']} "
        f"refused={c['refused']} dns={c['dns']}"
    )

print()
print("=== CLASSIFICATION (failed cycles) ===")
slow_exec = sum(1 for c in failed if c["exec_delay"] > 3)
only_refused = sum(
    1 for c in failed if c["refused"] > 0 and c["dns"] == 0 and c["tcp_ok"] == 0
)
refused_plus_dns = sum(1 for c in failed if c["refused"] > 0 and c["dns"] > 0)
only_dns = sum(
    1 for c in failed if c["dns"] > 0 and c["refused"] == 0 and c["tcp_ok"] == 0
)
tcp_but_failed = sum(1 for c in failed if c["tcp_ok"] > 0)
print(f"  exec_delay > 3s before PID assigned: {slow_exec}/{len(failed)}")
print(f"  only connection refused (port never open): {only_refused}/{len(failed)}")
print(f"  refused + dns timeouts: {refused_plus_dns}/{len(failed)}")
print(f"  only dns timeouts: {only_dns}/{len(failed)}")
print(f"  had TCP ok but still failed 10s: {tcp_but_failed}/{len(failed)}")

if failed:
    exec_delays = [c["exec_delay"] for c in failed]
    wait_windows = [c["wait_window"] for c in failed if c["wait_window"] is not None]
    print()
    print("=== TIMING STATS (failed) ===")
    print(f"  exec_delay: min={min(exec_delays):.0f}s max={max(exec_delays):.0f}s avg={sum(exec_delays)/len(exec_delays):.1f}s")
    if wait_windows:
        print(
            f"  wait_window (spawn->fail): min={min(wait_windows):.0f}s "
            f"max={max(wait_windows):.0f}s avg={sum(wait_windows)/len(wait_windows):.1f}s"
        )

print()
print("=== VERDICT ===")
if failed and only_refused + refused_plus_dns >= len(failed) * 0.7:
    print("CONFIRMED: Most failures show connection refused throughout the wait window —")
    print("server process was still starting (port not open) when the 10s timeout killed it.")
elif failed and slow_exec >= len(failed) * 0.5:
    print("CONFIRMED: Slow fork/exec delays consumed much of the startup budget before port checks.")
else:
    print("INCONCLUSIVE: Mixed failure modes; see breakdown above.")
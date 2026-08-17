#!/usr/bin/env bash
# Autonomous e2e gate for remote-agent terminal new/attach.
# Exit 0 only when the human journey works. Do not claim attach is fixed otherwise.
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

if ! command -v tty-watch >/dev/null 2>&1; then
  echo "Error: tty-watch not on PATH" >&2
  exit 1
fi

SCENE=$(mktemp -d "${TMPDIR:-/tmp}/verify-terminal-attach.XXXXXX")
export AI_CRITIC_HOME="$SCENE/homes"
export AI_CRITIC_NO_OPEN_BROWSER=1
export AI_CRITIC_TEST_SKIP_EXTENSION=1
export HOME="$SCENE/homes"
export TTY_WATCH_HOME="$SCENE/tty-watch"
mkdir -p "$AI_CRITIC_HOME" "$HOME" "$TTY_WATCH_HOME" "$SCENE/ws" "$SCENE/bin" "$SCENE/logs"
printf 'testpassword\n' > "$AI_CRITIC_HOME/server-credentials"
chmod 600 "$AI_CRITIC_HOME/server-credentials"

NEW_TTY=""
ATT_TTY=""
SERVER_PID=""
cleanup() {
  if [ -n "${ATT_TTY:-}" ]; then tty-watch kill "$ATT_TTY" 2>/dev/null || true; fi
  if [ -n "${NEW_TTY:-}" ]; then tty-watch kill "$NEW_TTY" 2>/dev/null || true; fi
  if [ -n "${SERVER_PID:-}" ]; then kill "$SERVER_PID" 2>/dev/null || true; fi
}
trap cleanup EXIT

fail() { echo "FAIL: $*" >&2; exit 1; }

parse_tty_id() {
  awk '{print $NF}' | tr -d '[:space:]'
}

echo "==> build server + remote-agent into $SCENE/bin"
GOSUMDB=off go build -mod=mod -o "$SCENE/bin/ai-critic-server" .
GOSUMDB=off go build -mod=mod -o "$SCENE/bin/remote-agent" ./cmd/remote-agent

PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
echo "==> start server :$PORT"
"$SCENE/bin/ai-critic-server" --quick-test --keep --port "$PORT" \
  --credentials-file "$AI_CRITIC_HOME/server-credentials" \
  --dir "$SCENE/ws" --no-event-bus-publish \
  >"$SCENE/logs/server.stdout" 2>"$SCENE/logs/server.stderr" &
SERVER_PID=$!
ok=0
for _ in $(seq 1 50); do
  if curl -sf "http://127.0.0.1:${PORT}/ping" >/dev/null; then ok=1; break; fi
  sleep 0.2
done
[ "$ok" = 1 ] || fail "server did not become ready ($(cat "$SCENE/logs/server.stderr"))"

RA=( "$SCENE/bin/remote-agent" --server "http://127.0.0.1:${PORT}" --token testpassword )
export PATH="$SCENE/bin:$PATH"

echo "==> 1. terminal new"
NEW_TTY=$(tty-watch run --detach --session-id "verify-new-$$" -- "${RA[@]}" terminal new --name verify-live | parse_tty_id)
[ -n "$NEW_TTY" ] || fail "tty-watch did not print a session id for terminal new"
sleep 1
NEW_SNAP=$(tty-watch snapshot "$NEW_TTY" 2>"$SCENE/logs/snap-new.err" || true)
echo "$NEW_SNAP" | tee "$SCENE/logs/snap-new.txt"
echo "$NEW_SNAP" | grep -q '\$' || fail "terminal new snapshot has no prompt: $NEW_SNAP"

LIST=$("${RA[@]}" terminal list)
echo "$LIST" | tee "$SCENE/logs/list-running.txt"
SID=$(echo "$LIST" | awk '/^session-/{print $1; exit}')
echo "$LIST" | grep -q running || fail "list has no running session"
[ -n "$SID" ] || fail "could not parse session id from list"

echo "==> 2. terminal attach $SID (must snapshot, not hang)"
ATT_TTY=$(tty-watch run --detach --session-id "verify-att-$$" -- "${RA[@]}" terminal attach "$SID" | parse_tty_id)
[ -n "$ATT_TTY" ] || fail "tty-watch did not print a session id for terminal attach"
sleep 1.2
set +e
ATT_SNAP=$(tty-watch snapshot "$ATT_TTY" 2>"$SCENE/logs/snap-att.err")
att_ec=$?
set -e
echo "$ATT_SNAP" | tee "$SCENE/logs/snap-att.txt"
cat "$SCENE/logs/snap-att.err" >>"$SCENE/logs/snap-att.txt" || true
[ "$att_ec" -eq 0 ] || fail "attach snapshot failed (hang): $(cat "$SCENE/logs/snap-att.err")"
echo "$ATT_SNAP" | grep -q '\$' || fail "attach snapshot has no prompt: $ATT_SNAP"
if printf '%s' "$ATT_SNAP" | grep -q $'\x1b\[?1049l'; then
  fail "attach snapshot contains alt-screen reset (Ctrl-L)"
fi
if printf '%s' "$ATT_SNAP" | grep -q $'\x1b\[2J'; then
  fail "attach snapshot contains erase display"
fi

echo "==> 3. type echo on attach TTY"
tty-watch send "$ATT_TTY" -- $'echo VERIFY_ATTACH_MARKER\r'
sleep 1.5
ECHO_SNAP=$(tty-watch snapshot "$ATT_TTY")
echo "$ECHO_SNAP" | tee "$SCENE/logs/snap-att-echo.txt"
echo "$ECHO_SNAP" | grep -q VERIFY_ATTACH_MARKER || fail "typed echo did not run: $ECHO_SNAP"

echo "==> 3b. Ctrl-] detaches attach client; session stays running"
tty-watch send "$ATT_TTY" -- $'\x1d'
DETACHED=""
for _ in $(seq 1 20); do
  DET_SNAP=$(tty-watch snapshot "$ATT_TTY" 2>/dev/null || true)
  if echo "$DET_SNAP" | grep -q 'detached'; then
    DETACHED=$DET_SNAP
    break
  fi
  sleep 0.15
done
echo "${DETACHED:-}" | tee "$SCENE/logs/snap-att-detach.txt"
[ -n "$DETACHED" ] || fail "Ctrl-] did not print detached"
LIST_KEEP=$("${RA[@]}" terminal list)
echo "$LIST_KEEP" | tee "$SCENE/logs/list-after-ctrl-bracket.txt"
echo "$LIST_KEEP" | grep -q running || fail "session should still be running after Ctrl-]"

echo "==> 3c. re-attach after Ctrl-]"
tty-watch kill "$ATT_TTY" 2>/dev/null || true
ATT_TTY=$(tty-watch run --detach --session-id "verify-att2-$$" -- "${RA[@]}" terminal attach "$SID" | parse_tty_id)
sleep 1.2
RE_SNAP=$(tty-watch snapshot "$ATT_TTY" 2>"$SCENE/logs/snap-reatt.err" || true)
echo "$RE_SNAP" | tee "$SCENE/logs/snap-reatt.txt"
echo "$RE_SNAP" | grep -q '\$' || fail "re-attach after Ctrl-] has no prompt: $RE_SNAP"
tty-watch send "$ATT_TTY" -- $'\x1d'
sleep 0.4

echo "==> 4. writer gone → attach must error, not hang"
tty-watch kill "$NEW_TTY" 2>/dev/null || true
NEW_TTY=""
sleep 0.8
LIST2=$("${RA[@]}" terminal list)
echo "$LIST2" | tee "$SCENE/logs/list-after-writer.txt"
echo "$LIST2" | grep -q exited || fail "session did not become exited after writer close"

set +e
DEAD_OUT=$(tty-watch run --detach --session-id "verify-dead-$$" -- "${RA[@]}" terminal attach "$SID")
DEAD_TTY=$(echo "$DEAD_OUT" | parse_tty_id)
DEAD_ERR=""
for _ in $(seq 1 15); do
  DEAD_SNAP=$(tty-watch snapshot "$DEAD_TTY" 2>/dev/null || true)
  if echo "$DEAD_SNAP" | grep -qi 'is exited'; then
    DEAD_ERR=$DEAD_SNAP
    break
  fi
  sleep 0.2
done
tty-watch kill "$DEAD_TTY" 2>/dev/null || true
set -e
[ -n "$DEAD_ERR" ] || fail "attach to exited session did not error within 3s (mute hang)"

echo "PASS verify-terminal-attach-e2e (scene $SCENE)"

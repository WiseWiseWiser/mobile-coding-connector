#!/usr/bin/env bash
# Hotspot diagnostics for remote-agent ws-proxy vpn.
# Run on the network you are testing (e.g. iPhone hotspot), not stable WiFi.
#
# Usage:
#   bash script/wsproxy-hotspot-diagnose.sh
#   bash script/wsproxy-hotspot-diagnose.sh --host ws-25b2a55939e4.xhd2015.xyz
#   WS_PROXY_HOST=ws-example.xhd2015.xyz bash script/wsproxy-hotspot-diagnose.sh --doctor
#
# Options:
#   --host HOST     ws-proxy public hostname (no https://)
#   --doctor        also run: remote-agent ws-proxy doctor
#   --skip-curl     skip TLS / WebSocket curl checks

set -euo pipefail

WS_HOST="${WS_PROXY_HOST:-}"
RUN_DOCTOR=false
SKIP_CURL=false

usage() {
	sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	--host)
		WS_HOST="${2:-}"
		shift 2
		;;
	--doctor)
		RUN_DOCTOR=true
		shift
		;;
	--skip-curl)
		SKIP_CURL=true
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		echo "unknown argument: $1" >&2
		usage >&2
		exit 2
		;;
	esac
done

normalize_host() {
	local raw="$1"
	raw="${raw#https://}"
	raw="${raw#http://}"
	raw="${raw%%/*}"
	raw="${raw%%/*}"
	printf '%s' "$raw"
}

strip_proxy_env() {
	env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY \
		-u http_proxy -u https_proxy -u all_proxy \
		-u NO_PROXY -u no_proxy "$@"
}

ip_prefix_matches() {
	local ip="$1"
	shift
	local prefix
	for prefix in "$@"; do
		case "$ip" in
		${prefix}*) return 0 ;;
		esac
	done
	return 1
}

is_likely_google_ipv4() {
	ip_prefix_matches "$1" 142.250. 142.251. 172.217. 172.218. 216.58. 216.59. 74.125.
}

is_likely_cloudflare_ipv4() {
	ip_prefix_matches "$1" \
		104.16. 104.17. 104.18. 104.19. 104.20. 104.21. 104.22. 104.23. \
		104.24. 104.25. 104.26. 104.27. \
		172.64. 172.65. 172.66. 172.67. 172.68. 172.69. 172.70. 172.71.
}

resolve_ws_host() {
	if [[ -n "$WS_HOST" ]]; then
		normalize_host "$WS_HOST"
		return 0
	fi

	local agent="${REMOTE_AGENT:-remote-agent}"
	if ! command -v "$agent" >/dev/null 2>&1; then
		echo ""
		return 1
	fi

	local status_out host
	status_out="$(strip_proxy_env "$agent" ws-proxy status 2>/dev/null || true)"
	host="$(printf '%s\n' "$status_out" | awk -F': ' '/^Public URL:/ { sub(/^[[:space:]]+/, "", $2); print $2 }' | head -1)"
	host="$(normalize_host "$host")"
	if [[ -n "$host" ]]; then
		printf '%s' "$host"
		return 0
	fi

	echo ""
	return 1
}

detect_hotspot_dns() {
	local gw ip

	gw="$(route -n get default 2>/dev/null | awk '/gateway:/{print $2}' | head -1)"
	if [[ -n "$gw" && "$gw" != "link#"* ]]; then
		printf '%s' "$gw"
		return 0
	fi

	while IFS= read -r ip; do
		[[ -z "$ip" ]] && continue
		[[ "$ip" == fe80:* ]] && continue
		[[ "$ip" == *:* ]] && continue
		printf '%s' "$ip"
		return 0
	done < <(scutil --dns 2>/dev/null | awk '/nameserver\[[0-9]+\] : / {print $3}')

	echo ""
	return 1
}

lookup_system_ipv4() {
	local host="$1"
	dscacheutil -q host -a name "$host" 2>/dev/null | awk '/ip_address:/{print $2}' | grep -E '^[0-9]+\.' | head -1
}

lookup_dig_ipv4() {
	local host="$1" server="$2"
	dig +short "$host" A @"$server" 2>/dev/null | grep -E '^[0-9]+\.' | head -1
}

print_dns_verdict() {
	local label="$1" ip="$2" kind="$3" optional="${4:-}"
	if [[ -z "$ip" ]]; then
		if [[ -n "$optional" ]]; then
			printf '  %-22s (no IPv4 answer)  [skipped]\n' "$label"
			return 0
		fi
		printf '  %-22s (no IPv4 answer)\n' "$label"
		return 1
	fi
	case "$kind" in
	google)
		if is_likely_google_ipv4 "$ip"; then
			printf '  %-22s %s  [ok]\n' "$label" "$ip"
			return 0
		fi
		printf '  %-22s %s  [POLLUTED — not a Google IP]\n' "$label" "$ip"
		return 1
		;;
	cloudflare)
		if is_likely_cloudflare_ipv4 "$ip"; then
			printf '  %-22s %s  [ok]\n' "$label" "$ip"
			return 0
		fi
		printf '  %-22s %s  [SUSPICIOUS — expected Cloudflare]\n' "$label" "$ip"
		return 1
		;;
	esac
}

section() {
	echo
	echo "=== $1 ==="
}

failures=0
record_fail() { failures=$((failures + 1)); }

section "Environment"
ACTIVE_IF="$(route -n get default 2>/dev/null | awk '/interface:/{print $2}' | head -1 || true)"
echo "  active interface:     ${ACTIVE_IF:-unknown}"
HOTSPOT_DNS="$(detect_hotspot_dns || true)"
echo "  default gateway DNS:  ${HOTSPOT_DNS:-unknown}"

WS_HOST="$(resolve_ws_host || true)"
if [[ -z "$WS_HOST" ]]; then
	echo "  ws-proxy host:        MISSING" >&2
	echo >&2
	echo "Could not resolve ws-proxy hostname." >&2
	echo "  - Ensure remote-agent can reach your ai-critic server, or" >&2
	echo "  - Pass --host ws-YOURSUBDOMAIN.xhd2015.xyz (hostname only, no https://)" >&2
	exit 1
fi
echo "  ws-proxy host:        $WS_HOST"

section "1) DNS pollution probe (www.google.com)"
SYS_GOOGLE="$(lookup_system_ipv4 www.google.com || true)"
TRUSTED_GOOGLE="$(lookup_dig_ipv4 www.google.com 8.8.8.8 || true)"
HOTSPOT_GOOGLE=""
if [[ -n "$HOTSPOT_DNS" ]]; then
	HOTSPOT_GOOGLE="$(lookup_dig_ipv4 www.google.com "$HOTSPOT_DNS" || true)"
fi
print_dns_verdict "system (libc):" "$SYS_GOOGLE" google || record_fail
print_dns_verdict "direct 8.8.8.8:" "$TRUSTED_GOOGLE" google || record_fail
if [[ -n "$HOTSPOT_DNS" ]]; then
	print_dns_verdict "hotspot gateway:" "$HOTSPOT_GOOGLE" google optional || record_fail
else
	echo "  hotspot gateway:      (skipped — no IPv4 gateway detected)"
fi
if [[ -n "$SYS_GOOGLE" && -n "$TRUSTED_GOOGLE" && "$SYS_GOOGLE" != "$TRUSTED_GOOGLE" ]]; then
	if ! is_likely_google_ipv4 "$SYS_GOOGLE" || ! is_likely_google_ipv4 "$TRUSTED_GOOGLE"; then
		echo "  note: system and 8.8.8.8 disagree with pollution — use vpn --http-only --dns-hijack"
	fi
fi
if [[ -n "$TRUSTED_GOOGLE" ]] && ! is_likely_google_ipv4 "$TRUSTED_GOOGLE"; then
	echo "  note: even 8.8.8.8 is poisoned on this hotspot; --dns-hijack resolves via ws-proxy"
fi

section "2) ws-proxy hostname DNS ($WS_HOST)"
SYS_WS="$(lookup_system_ipv4 "$WS_HOST" || true)"
TRUSTED_WS="$(lookup_dig_ipv4 "$WS_HOST" 8.8.8.8 || true)"
HOTSPOT_WS=""
if [[ -n "$HOTSPOT_DNS" ]]; then
	HOTSPOT_WS="$(lookup_dig_ipv4 "$WS_HOST" "$HOTSPOT_DNS" || true)"
fi
print_dns_verdict "system (libc):" "$SYS_WS" cloudflare || record_fail
print_dns_verdict "direct 8.8.8.8:" "$TRUSTED_WS" cloudflare || record_fail
if [[ -n "$HOTSPOT_DNS" ]]; then
	print_dns_verdict "hotspot gateway:" "$HOTSPOT_WS" cloudflare optional || record_fail
else
	echo "  hotspot gateway:      (skipped)"
fi

if [[ -z "$SYS_WS" && -z "$TRUSTED_WS" ]]; then
	echo "  hint: if you passed https:// in --host, drop the scheme; this script strips it automatically"
	record_fail
fi

if [[ -n "$SKIP_CURL" ]]; then
	section "3) Direct reachability (skipped — --skip-curl)"
else
	section "3) Direct reachability (no VMess, no TUN)"
	if strip_proxy_env curl -4 -m 12 -sS -o /dev/null -w "  TLS handshake:        HTTP %{http_code}\n" "https://${WS_HOST}/"; then
		:
	else
		echo "  TLS handshake:        FAILED (timeout or connection error)" >&2
		record_fail
	fi
	ws_code="$(strip_proxy_env curl -4 -m 15 -sS -o /dev/null -w '%{http_code}' "https://${WS_HOST}/ws" 2>/dev/null || echo 000)"
	if [[ "$ws_code" == "400" ]]; then
		printf '  WebSocket /ws:        HTTP %s  [ok]\n' "$ws_code"
	elif [[ "$ws_code" == "000" ]]; then
		echo "  WebSocket /ws:        FAILED (timeout or connection error)" >&2
		record_fail
	else
		echo "  WebSocket /ws:        HTTP $ws_code  [expected 400 from xray]" >&2
		record_fail
	fi
fi

if [[ "$RUN_DOCTOR" == true ]]; then
	section "4) remote-agent ws-proxy doctor"
	agent="${REMOTE_AGENT:-remote-agent}"
	if ! command -v "$agent" >/dev/null 2>&1; then
		echo "  remote-agent not found in PATH" >&2
		record_fail
	else
		if ! strip_proxy_env "$agent" ws-proxy doctor; then
			echo "  doctor reported failures (see Client checks above)" >&2
			record_fail
		fi
	fi
else
	section "4) VMess doctor (skipped)"
	echo "  run with --doctor to include: remote-agent ws-proxy doctor"
fi

section "Summary"
if [[ "$failures" -eq 0 ]]; then
	echo "  PASS — hotspot path looks healthy from these checks"
	echo "  If vpn --http-only still fails after TUN comes up, the issue is likely TUN routing timing."
	exit 0
fi

echo "  FAIL — $failures check group(s) failed on this network"
echo "  Common fixes:"
echo "    - DNS polluted → remote-agent ws-proxy vpn --http-only --dns-hijack"
echo "    - ws-proxy host unknown → bash $0 --host YOUR.ws-subdomain.xhd2015.xyz"
echo "    - curl timeouts with good DNS → hotspot may be blocking Cloudflare; try again or different network"
exit 1
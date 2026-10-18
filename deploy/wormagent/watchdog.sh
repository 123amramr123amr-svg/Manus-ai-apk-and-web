#!/usr/bin/env bash
# watchdog.sh — يراقب بوابة WormAgent ويعيد تشغيلها تلقائيًا إذا توقفت أو علّقت.
#
# الاستخدام:
#   ./watchdog.sh            # يعمل في الخلفية حتى تُوقفه
#   ./watchdog.sh --stop     # إيقاف المراقب
#   ./watchdog.sh --status   # حالة المراقب والبوابة
#
# متغيّرات: WATCHDOG_INTERVAL (ثواني بين الفحوص، افتراضي 30) · WORMAGENT_PORT · WORMAGENT_DIR · WORMAGENT_STATE_DIR
#
# ⚠️ حدود: هذا يحمي من **توقّف العملية/تعليقها** فقط. لا يمكنه منع إيقاف/إعادة تدوير
# البيئة نفسها (حاوية/سيرفر). للتشغيل 24/7 استخدم جهازك أو VPS + systemd (wormagent.service).

set -euo pipefail

INTERVAL="${WATCHDOG_INTERVAL:-30}"
PORT="${WORMAGENT_PORT:-18789}"
SRC_DIR="${WORMAGENT_DIR:-$HOME/wormagent}"
STATE_DIR="${WORMAGENT_STATE_DIR:-$HOME/.wormagent}"
PID_FILE="$STATE_DIR/watchdog.pid"
LOG="$STATE_DIR/watchdog.log"
HERE="$(cd "$(dirname "$0")" && pwd)"

log() { printf '%s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >> "$LOG"; }
alive() { curl -fsS -m 5 -o /dev/null "http://127.0.0.1:$PORT/" 2>/dev/null; }

case "${1:-}" in
  --stop)
    if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
      kill "$(cat "$PID_FILE")" && echo "watchdog stopped"
    else
      echo "watchdog not running"
    fi
    rm -f "$PID_FILE"
    exit 0 ;;
  --status)
    if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
      echo "watchdog: running (pid $(cat "$PID_FILE"))"
    else
      echo "watchdog: not running"
    fi
    if alive; then echo "gateway: up"; else echo "gateway: DOWN"; fi
    [[ -f "$LOG" ]] && { echo "--- last log lines ---"; tail -5 "$LOG"; }
    exit 0 ;;
esac

if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
  echo "watchdog already running (pid $(cat "$PID_FILE"))"
  exit 0
fi

mkdir -p "$STATE_DIR"
echo $$ > "$PID_FILE"
log "watchdog started (interval=${INTERVAL}s, port=$PORT, dir=$SRC_DIR)"

while true; do
  if ! alive; then
    log "gateway DOWN -> restarting"
    ( cd "$SRC_DIR" && setsid nohup node wormagent.mjs gateway >> "$STATE_DIR/gateway.log" 2>&1 < /dev/null & )
    # انتظر حتى تستجيب (حتى 60 ثانية)
    for _ in $(seq 1 20); do sleep 3; alive && break; done
    if alive; then log "gateway UP again"; else log "restart failed (needs manual check)"; fi
  fi
  sleep "$INTERVAL"
done

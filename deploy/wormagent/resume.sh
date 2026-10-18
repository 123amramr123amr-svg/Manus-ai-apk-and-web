#!/usr/bin/env bash
# resume.sh — استئناف سريع (أمر واحد) بعد إعادة تشغيل البيئة:
#   1) يتحقق من الاعتماديات (ينادي setup.sh إن لزم)  2) يشغّل البوابة إن كانت متوقفة
#   3) يشغّل المراقب (watchdog) إن لم يكن يعمل        4) يطبع الحالة
#
# الاستخدام:  ./resume.sh        (من داخل مجلد المستودع أو عبر المسار الكامل)
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
SRC_DIR="${WORMAGENT_DIR:-$(cd "$HERE/../.." && pwd)/wormagent}"
STATE_DIR="${WORMAGENT_STATE_DIR:-$HOME/.wormagent}"
PORT="${WORMAGENT_PORT:-18789}"

log() { printf '\033[1;36m[resume]\033[0m %s\n' "$*"; }
up() { curl -fsS -m 5 -o /dev/null "http://127.0.0.1:$PORT/" 2>/dev/null; }

if [[ ! -f "$SRC_DIR/wormagent.mjs" ]]; then
  log "لم أجد الكود في $SRC_DIR — حدّد المسار: WORMAGENT_DIR=/path/to/wormagent"
  exit 1
fi

if [[ ! -f "$SRC_DIR/node_modules/.package-lock.json" ]]; then
  log "الاعتماديات غير مثبّتة → تثبيت"
  WORMAGENT_DIR="$SRC_DIR" bash "$HERE/setup.sh"
fi

if up; then
  log "البوابة تعمل بالفعل (:$PORT)"
else
  log "تشغيل البوابة"
  WORMAGENT_DIR="$SRC_DIR" bash "$HERE/setup.sh" --dir "$SRC_DIR" --skip-install | grep -E 'جاهزة|تعمل|تنبيه' || true
fi

PID_FILE="$STATE_DIR/watchdog.pid"
watchdog_running() { [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; }

if watchdog_running; then
  log "المراقب يعمل بالفعل (pid $(cat "$PID_FILE"))"
else
  log "تشغيل المراقب (watchdog)"
  ( setsid env WORMAGENT_DIR="$SRC_DIR" WORMAGENT_PORT="$PORT" nohup "$HERE/watchdog.sh" >/dev/null 2>&1 < /dev/null & )
  sleep 3
fi

log "الحالة النهائية:"
bash "$HERE/watchdog.sh" --status || true

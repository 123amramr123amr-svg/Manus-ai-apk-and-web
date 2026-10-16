#!/usr/bin/env bash
# setup.sh — تثبيت وتشغيل WormAgent (البوابة + واجهة التحكم) على جهازك أو سيرفرك.
#
# الاستخدام:
#   ./setup.sh                                    # تثبيت + تشغيل محلي (127.0.0.1:18789)
#   ./setup.sh --telegram "123:ABC" --bind lan    # مع قناة تلجرام + فتح الشبكة المحلية
#   ./setup.sh --no-start                         # تثبيت وإعداد فقط بدون تشغيل
#   ./setup.sh --skip-install                     # تجاهل npm install (الاعتماديات موجودة)
#   ./setup.sh --port 19000 --dir /opt/wormagent  # منفذ/مجلد مخصصان
#   ./setup.sh --stop                             # إيقاف البوابة التي شغّلها هذا السكربت
#
# متغيرات البيئة المدعومة:
#   WORMAGENT_REPO_URL  رابط المستودع (افتراضي: BlackXV2vip/wormagent)
#   WORMAGENT_DIR       مجلد الكود       WORMAGENT_PORT  منفذ البوابة
#   WORMAGENT_BIND      loopback|lan|tailnet   WORMAGENT_STATE_DIR  مجلد الإعدادات
#   TELEGRAM_BOT_TOKEN  مفتاح بوت تلجرام من @BotFather

set -euo pipefail

REPO_URL="${WORMAGENT_REPO_URL:-https://github.com/BlackXV2vip/wormagent.git}"
SRC_DIR="${WORMAGENT_DIR:-$HOME/wormagent}"
STATE_DIR="${WORMAGENT_STATE_DIR:-$HOME/.wormagent}"
PORT="${WORMAGENT_PORT:-18789}"
BIND="${WORMAGENT_BIND:-loopback}"
TELEGRAM_TOKEN="${TELEGRAM_BOT_TOKEN:-}"
DO_START=1
DO_INSTALL=1
DO_STOP=0

log() { printf '\033[1;36m[wormagent]\033[0m %s\n' "$*"; }
die() { printf '\033[1;31m[wormagent] خطأ:\033[0m %s\n' "$*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dir) SRC_DIR="$2"; shift 2 ;;
    --port) PORT="$2"; shift 2 ;;
    --bind) BIND="$2"; shift 2 ;;
    --telegram) TELEGRAM_TOKEN="$2"; shift 2 ;;
    --state-dir) STATE_DIR="$2"; shift 2 ;;
    --no-start) DO_START=0; shift ;;
    --skip-install) DO_INSTALL=0; shift ;;
    --stop) DO_STOP=1; shift ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    *) die "خيار غير معروف: $1 (استخدم --help)" ;;
  esac
done

PID_FILE="$STATE_DIR/gateway.pid"

if [[ "$DO_STOP" == 1 ]]; then
  STOPPED=0
  if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    kill "$(cat "$PID_FILE")" 2>/dev/null && STOPPED=1
  fi
  # البوابة تعمل كعملية فرعية باسم wormagent-gateway — نُنهيها أيضًا
  for pid in $(pgrep -f 'wormagent[-]gateway' 2>/dev/null || true); do
    kill "$pid" 2>/dev/null && STOPPED=1
  done
  rm -f "$PID_FILE"
  if [[ "$STOPPED" == 1 ]]; then log "أُوقفت البوابة."; else log "لا توجد بوابة شغّالة."; fi
  for _ in $(seq 1 15); do
    curl -fsS -m 2 -o /dev/null "http://127.0.0.1:$PORT/" 2>/dev/null || break
    sleep 1
  done
  exit 0
fi

# 1) فحص Node.js
command -v node >/dev/null 2>&1 || die "Node.js غير مثبّت. المطلوب Node 22.16+ (الموصى به 24)."
node -e 'const [M,m]=process.versions.node.split(".").map(Number);process.exit((M>22||(M===22&&m>=16))?0:1)' \
  || die "إصدار Node الحالي $(node -v) قديم — المطلوب 22.16 أو أحدث."
command -v npm >/dev/null 2>&1 || die "npm غير مثبّت."

# 2) جلب الكود (clone أو تحديث)
if [[ -d "$SRC_DIR/.git" ]]; then
  log "تحديث $SRC_DIR"
  git -C "$SRC_DIR" pull --ff-only 2>/dev/null || log "(تعذّر التحديث — نكمل بالنسخة الحالية)"
elif [[ -f "$SRC_DIR/wormagent.mjs" ]]; then
  log "استخدام مجلد موجود: $SRC_DIR"
else
  command -v git >/dev/null 2>&1 || die "git غير مثبّت (مطلوب للاستنساخ)."
  log "استنساخ $REPO_URL → $SRC_DIR"
  git clone --depth 1 "$REPO_URL" "$SRC_DIR"
fi
[[ -f "$SRC_DIR/wormagent.mjs" ]] || die "الملف wormagent.mjs غير موجود في $SRC_DIR"
cd "$SRC_DIR"

# 3) تثبيت الاعتماديات
if [[ "$DO_INSTALL" == 1 ]]; then
  if [[ -f node_modules/.package-lock.json ]]; then
    log "الاعتماديات مثبّتة مسبقًا — تخطّي التثبيت."
  else
    log "تثبيت الاعتماديات (قد يستغرق عدة دقائق)"
    npm install --omit=dev --legacy-peer-deps --no-audit --no-fund
  fi
fi

# سكربت prepare داخل الحزمة ينفّذ: git config core.hooksPath git-hooks
# إن كان المجلد داخل مستودع git آخر (بدون .git خاص به) فسيُعدَّل إعداد ذلك المستودع — نُعيده لحالته.
if [[ ! -d "$SRC_DIR/.git" ]]; then
  TOP="$(git -C "$SRC_DIR" rev-parse --show-toplevel 2>/dev/null || true)"
  if [[ -n "$TOP" && "$(git -C "$SRC_DIR" config --local --get core.hooksPath 2>/dev/null || true)" == "git-hooks" ]]; then
    git -C "$SRC_DIR" config --local --unset core.hooksPath || true
    log "أُعيد core.hooksPath في $TOP إلى حالته الأصلية."
  fi
fi

# 4) كتابة الإعدادات
mkdir -p "$STATE_DIR"
CFG="$STATE_DIR/wormagent.json"
EXISTING_TOKEN=""
if [[ -f "$CFG" ]]; then
  EXISTING_TOKEN="$(node -e "try{const c=require('$CFG');process.stdout.write((c.gateway&&c.gateway.auth&&c.gateway.auth.token)||'')}catch(e){}" 2>/dev/null || true)"
fi
TOKEN="${EXISTING_TOKEN:-$(node -e 'process.stdout.write(require("crypto").randomBytes(24).toString("hex"))')}"

set_cfg() {
  local out
  if ! out="$(node wormagent.mjs config set "$1" "$2" 2>&1)"; then
    printf '%s\n' "$out" | grep -v 'Config overwrite' >&2
    die "فشل ضبط الإعداد $1"
  fi
  printf '%s\n' "$out" | grep -E '^(Updated|Removed|Unknown|Error|Invalid)' || true
}

log "كتابة الإعدادات في $CFG"
set_cfg gateway.mode local
set_cfg gateway.bind "$BIND"
set_cfg gateway.port "$PORT"
set_cfg gateway.auth.mode token
set_cfg gateway.auth.token "$TOKEN"

# تفعيل الوكلاء الفرعيين (sub-agents): تفرّع متداخل + حدود التزامن + مهلة
set_cfg agents.defaults.subagents.maxSpawnDepth 2
set_cfg agents.defaults.subagents.maxChildrenPerAgent 5
set_cfg agents.defaults.subagents.runTimeoutSeconds 900
set_cfg agents.defaults.subagents.archiveAfterMinutes 60

# الوصول من خارج 127.0.0.1 يحتاج أصل المتصفح (Origin) مسموحًا به صراحةً.
if [[ "$BIND" != "loopback" ]]; then
  LAN_IP="$(hostname -I 2>/dev/null | awk '{print $1}' || true)"
  if [[ -n "${LAN_IP:-}" ]]; then
    set_cfg gateway.controlUi.allowedOrigins "[\"http://$LAN_IP:$PORT\"]"
    log "سُمح بأصل المتصفح http://$LAN_IP:$PORT"
  else
    log "تنبيه: لم أتمكّن من اكتشاف IP — أضف أصل المتصفح يدويًا في gateway.controlUi.allowedOrigins"
  fi
fi

if [[ -n "$TELEGRAM_TOKEN" ]]; then
  log "تفعيل قناة تلجرام"
  set_cfg channels.telegram.enabled true
  set_cfg channels.telegram.botToken "$TELEGRAM_TOKEN"
  set_cfg channels.telegram.dmPolicy pairing
  # أزرار تبديل النماذج داخل تلجرام (/model و /models)
  set_cfg channels.telegram.capabilities.inlineButtons all
  set_cfg commands.native auto
  set_cfg commands.text true
fi

# 5) التشغيل
if [[ "$DO_START" == 1 ]]; then
  if curl -fsS -m 3 "http://127.0.0.1:$PORT/" 2>/dev/null | grep -q 'WormAgent'; then
    log "البوابة تعمل بالفعل على المنفذ $PORT — لم أُشغّل نسخة ثانية."
  else
    log "تشغيل البوابة (log: $STATE_DIR/gateway.log)"
    setsid nohup node wormagent.mjs gateway > "$STATE_DIR/gateway.log" 2>&1 < /dev/null &
    READY=0
    for _ in $(seq 1 40); do
      sleep 1
      if curl -fsS -o /dev/null "http://127.0.0.1:$PORT/" 2>/dev/null; then READY=1; break; fi
    done
    GW_PID="$(pgrep -f 'wormagent[-]gateway' 2>/dev/null | tail -1 || true)"
    echo "${GW_PID:-$!}" > "$PID_FILE"
    if [[ "$READY" == 1 ]]; then log "البوابة جاهزة."; else log "تنبيه: لم تستجب البوابة بعد 40 ثانية — راجع $STATE_DIR/gateway.log"; fi
  fi
fi

cat <<EOF

──────────────────────── الخلاصة ────────────────────────
واجهة التحكم (Control UI)  : http://127.0.0.1:$PORT/
مفتاح الدخول (Gateway token): $TOKEN
ملف الإعدادات              : $CFG
سجل التشغيل                : $STATE_DIR/gateway.log
إيقاف البوابة              : $0 --stop
─────────────────────────────────────────────────────────
EOF

if [[ -n "$TELEGRAM_TOKEN" ]]; then
  cat <<EOF

تلجرام: أرسل رسالة للبوت من حسابك، ثم اعتمد الاقتران:
  cd $SRC_DIR && node wormagent.mjs pairing list telegram
  cd $SRC_DIR && node wormagent.mjs pairing approve telegram <CODE>
EOF
fi

cat <<EOF

ملاحظة: المحادثة تحتاج مزوّد موديل (LLM) — أضف مفتاحك:
  cd $SRC_DIR && node wormagent.mjs onboard
EOF


#!/usr/bin/env bash
# add-model-key.sh — إضافة مفتاح مزوّد موديل (LLM) إلى WormAgent وإعادة تشغيل البوابة.
#
# الاستخدام:
#   ./add-model-key.sh gemini    "AIza..."             # Google Gemini (فيه طبقة مجانية)
#   ./add-model-key.sh openai    "sk-..."              # OpenAI
#   ./add-model-key.sh anthropic "sk-ant-..."          # Anthropic (Claude)
#   ./add-model-key.sh openrouter "sk-or-..." [MODEL]  # OpenRouter (الافتراضي موديل مجاني)
#   ./add-model-key.sh mistral   "..."                 # Mistral
#   ./add-model-key.sh custom    "KEY" "BASE_URL" "MODEL_ID"   # أي مزوّد متوافق مع OpenAI
#   ./add-model-key.sh ollama    "llama3.2:3b"         # مزوّد محلي بدون مفتاح
#
# متغيرات: WORMAGENT_DIR (افتراضي ~/wormagent)، WORMAGENT_STATE_DIR (افتراضي ~/.wormagent)

set -euo pipefail

PROVIDER="${1:-}"; KEY="${2:-}"; ARG3="${3:-}"; ARG4="${4:-}"
SRC_DIR="${WORMAGENT_DIR:-$HOME/wormagent}"
STATE_DIR="${WORMAGENT_STATE_DIR:-$HOME/.wormagent}"
CFG="$STATE_DIR/wormagent.json"
HERE="$(cd "$(dirname "$0")" && pwd)"

usage() { sed -n '2,16p' "$0"; }

if [[ -z "$PROVIDER" ]]; then usage; exit 1; fi
if [[ ! -f "$SRC_DIR/wormagent.mjs" ]]; then
  echo "خطأ: شغّل setup.sh أولًا — المجلد $SRC_DIR لا يحتوي wormagent.mjs" >&2; exit 1
fi
cd "$SRC_DIR"

case "$PROVIDER" in
  gemini|google)    AUTH=(--auth-choice gemini-api-key --gemini-api-key "$KEY");           MODEL="google/gemini-3-flash-preview" ;;
  openai)           AUTH=(--auth-choice openai-api-key --openai-api-key "$KEY");           MODEL="openai/gpt-5.2" ;;
  anthropic|claude) AUTH=(--auth-choice apiKey --anthropic-api-key "$KEY");                MODEL="anthropic/claude-opus-4-6" ;;
  openrouter)       AUTH=(--auth-choice openrouter-api-key --openrouter-api-key "$KEY");   MODEL="${ARG3:-openrouter/meta-llama/llama-3.3-70b-instruct:free}" ;;
  mistral)          AUTH=(--auth-choice mistral-api-key --mistral-api-key "$KEY");         MODEL="mistral/mistral-large-latest" ;;
  custom)
    PROV_ID="${CUSTOM_PROVIDER_ID:-$(printf '%s' "$ARG3" | sed -E 's#^https?://##; s#[/:].*$##; s#^api\.##; s#[^A-Za-z0-9]+#-#g; s#-$##')}"
    AUTH=(--auth-choice custom-api-key --custom-api-key "$KEY" --custom-base-url "$ARG3" --custom-model-id "$ARG4" --custom-provider-id "$PROV_ID" --custom-compatibility openai)
    MODEL="$PROV_ID/$ARG4"
    ;;
  ollama)           AUTH=(--auth-choice ollama --custom-model-id "$KEY");                  MODEL="ollama/$KEY" ;;
  *) echo "مزوّد غير معروف: $PROVIDER" >&2; usage; exit 1 ;;
esac

if [[ "$PROVIDER" != "ollama" && -z "$KEY" ]]; then
  echo "خطأ: لم تُمرّر المفتاح." >&2; usage; exit 1
fi

# نُبقي إعدادات البوابة الحالية كما هي (منفذ/نطاق/توكن)
read_cfg() { node -e "try{const c=require('$CFG');const v=$1;process.stdout.write(v==null?'':String(v))}catch(e){process.stdout.write('')}" 2>/dev/null || true; }
PORT="$(read_cfg "c.gateway&&c.gateway.port")"; PORT="${PORT:-18789}"
BIND="$(read_cfg "c.gateway&&c.gateway.bind")"; BIND="${BIND:-loopback}"
TOKEN="$(read_cfg "c.gateway&&c.gateway.auth&&c.gateway.auth.token")"

echo "[model] إضافة مفتاح $PROVIDER ..."
GATEWAY_ARGS=(--gateway-port "$PORT" --gateway-bind "$BIND")
if [[ -n "$TOKEN" ]]; then GATEWAY_ARGS+=(--gateway-auth token --gateway-token "$TOKEN"); fi

node wormagent.mjs onboard --non-interactive --mode local \
  "${AUTH[@]}" --secret-input-mode plaintext --no-install-daemon --accept-risk \
  "${GATEWAY_ARGS[@]}" 2>&1 | grep -viE 'Config overwrite' | tail -10 || true

echo "[model] ضبط الموديل الافتراضي: $MODEL"
node wormagent.mjs models set "$MODEL" 2>&1 | grep -E '^(Updated|Default model|Error|Invalid)' || true

echo "[model] إعادة تشغيل البوابة لتطبيق الإعدادات ..."
bash "$HERE/setup.sh" --dir "$SRC_DIR" --stop >/dev/null 2>&1 || true
bash "$HERE/setup.sh" --dir "$SRC_DIR" --skip-install 2>&1 | grep -E 'جاهزة|تنبيه|مفتاح الدخول' || true

cat <<EOF

تم. اختبر الآن بدورة وكيل واحدة:
  cd $SRC_DIR && node wormagent.mjs agent -m 'قل مرحبا' --agent main
أو أرسل رسالة للبوت على تلجرام.
EOF

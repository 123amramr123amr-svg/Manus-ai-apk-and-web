#!/usr/bin/env bash
# add-provider-models.sh — إضافة **كل** موديلات مزوّد متوافق مع OpenAI بأمر واحد.
#
# الاستخدام:
#   ./add-provider-models.sh <BASE_URL> <API_KEY> [PROVIDER_ID] [--dry-run] [--context N] [--no-restart]
#
# مثال:
#   ./add-provider-models.sh https://codecraftapi.com/v1 "cc_xxxx" codecraftapi
#   ./add-provider-models.sh https://codecraftapi.com/v1 "cc_xxxx" --dry-run   # عرض فقط بدون تعديل
#
# الخطوات:
#   1) GET {BASE_URL}/models  → قائمة الموديلات
#   2) استبعاد الموديلات المسجّلة مسبقًا لنفس المزوّد
#   3) تسجيل المزوّد (أول موديل) ثم كتابة كل الموديلات في models.providers.<PROVIDER_ID>.models
#   4) إعادة تشغيل البوابة

set -euo pipefail

BASE_URL="${1:-}"; API_KEY="${2:-}"; PROV_ARG="${3:-}"
DRY=0; CTX=64000; RESTART=1
for a in "$@"; do
  case "$a" in
    --dry-run) DRY=1 ;;
    --no-restart) RESTART=0 ;;
    --context) : ;;
  esac
done
# --context N
prev=""
for a in "$@"; do
  if [[ "$prev" == "--context" ]]; then CTX="$a"; fi
  prev="$a"
done
[[ "$PROV_ARG" == --* ]] && PROV_ARG=""

SRC_DIR="${WORMAGENT_DIR:-$HOME/wormagent}"
STATE_DIR="${WORMAGENT_STATE_DIR:-$HOME/.wormagent}"
CFG="$STATE_DIR/wormagent.json"
HERE="$(cd "$(dirname "$0")" && pwd)"

if [[ -z "$BASE_URL" || -z "$API_KEY" ]]; then sed -n '2,16p' "$0"; exit 1; fi
[[ -f "$SRC_DIR/wormagent.mjs" ]] || { echo "خطأ: شغّل setup.sh أولًا." >&2; exit 1; }
cd "$SRC_DIR"

PROV_ID="${PROV_ARG:-$(printf '%s' "$BASE_URL" | sed -E 's#^https?://##; s#[/:].*$##; s#^api\.##; s#[^A-Za-z0-9]+#-#g; s#-$##')}"

echo "[models] جلب القائمة من $BASE_URL/models"
RAW="$(curl -sS -m 30 "$BASE_URL/models" -H "Authorization: Bearer $API_KEY" || true)"
MODELS="$(printf '%s' "$RAW" | python3 -c '
import json,sys
raw=sys.stdin.read()
try:
    d=json.loads(raw)
except Exception:
    sys.exit(2)
ids=[m.get("id") for m in d.get("data",[]) if m.get("id")]
print("\n".join(ids))
')" || { echo "تعذّر قراءة قائمة الموديلات. رد المزوّد:" >&2; printf '%s\n' "$RAW" | head -c 400 >&2; exit 1; }

TOTAL="$(printf '%s\n' "$MODELS" | grep -c . || true)"
echo "[models] المزوّد: $PROV_ID — عدد الموديلات المتاحة: $TOTAL"

# الموديلات المسجّلة مسبّقًا لنفس المزوّد
EXISTING="$(node -e "try{const c=require('$CFG');const p=(c.models&&c.models.providers&&c.models.providers['$PROV_ID']);const ms=(p&&p.models)||[];process.stdout.write(ms.map(m=>m.id).join('\n'))}catch(e){}" 2>/dev/null || true)"

NEW="$(printf '%s\n' "$MODELS" | grep . | while read -r m; do
  if ! printf '%s\n' "$EXISTING" | grep -Fxq "$m"; then echo "$m"; fi
done)"

NEW_COUNT="$(printf '%s\n' "$NEW" | grep -c . || true)"
echo "[models] موديلات جديدة سيتم إضافتها: $NEW_COUNT (من أصل $TOTAL)"
if [[ "$NEW_COUNT" != "0" ]]; then printf '%s\n' "$NEW" | head -40 | sed 's/^/   - /'; fi

if [[ "$NEW_COUNT" == "0" ]]; then echo "[models] لا يوجد جديد — انتهى."; exit 0; fi
if [[ "$DRY" == 1 ]]; then echo "[models] وضع العرض فقط (--dry-run) — لم يُعدَّل أي شيء."; exit 0; fi

FIRST="$(printf '%s\n' "$NEW" | head -1)"
TOKEN="$(node -e "try{process.stdout.write(require('$CFG').gateway.auth.token||'')}catch(e){}" 2>/dev/null || true)"
PORT="$(node -e "try{process.stdout.write(String(require('$CFG').gateway.port))}catch(e){process.stdout.write('18789')}" 2>/dev/null || echo 18789)"
BIND="$(node -e "try{process.stdout.write(require('$CFG').gateway.bind||'loopback')}catch(e){process.stdout.write('loopback')}" 2>/dev/null || echo loopback)"

echo "[models] تسجيل المزوّد (الموديل الأول: $FIRST) ..."
PREV_DEFAULT="$(node -e "try{process.stdout.write(((require('$CFG').agents||{}).defaults||{}).model?.primary||'')}catch(e){}" 2>/dev/null || true)"
GATEWAY_ARGS=(--gateway-port "$PORT" --gateway-bind "$BIND")
[[ -n "$TOKEN" ]] && GATEWAY_ARGS+=(--gateway-auth token --gateway-token "$TOKEN")
node wormagent.mjs onboard --non-interactive --mode local \
  --auth-choice custom-api-key --custom-api-key "$API_KEY" --custom-base-url "$BASE_URL" \
  --custom-model-id "$FIRST" --custom-provider-id "$PROV_ID" --custom-compatibility openai \
  --secret-input-mode plaintext --no-install-daemon --accept-risk "${GATEWAY_ARGS[@]}" \
  2>&1 | grep -viE 'Config overwrite|groupPolicy|allowFrom|sender IDs|^[│├╯╮─◇┌└ ]' | tail -3 || true

# onboard يضبط الموديل الافتراضي على الموديل الجديد — نُعيد الافتراضي السابق
if [[ -n "$PREV_DEFAULT" ]]; then
  node wormagent.mjs models set "$PREV_DEFAULT" >/dev/null 2>&1 || true
  echo "[models] أُعيد الموديل الافتراضي إلى: $PREV_DEFAULT"
fi

echo "[models] كتابة كل الموديلات ($NEW_COUNT) في الإعداد ..."
JSON="$(printf '%s' "$NEW" | CTX="$CTX" python3 -c '
import json,sys,os
ctx=int(os.environ.get("CTX","64000"))
ids=[l.strip() for l in sys.stdin.read().splitlines() if l.strip()]
out=[{"id":i,"name":i+" (Custom Provider)","reasoning":False,"input":["text"],
      "cost":{"input":0,"output":0,"cacheRead":0,"cacheWrite":0},
      "contextWindow":ctx,"maxTokens":8192} for i in ids]
print(json.dumps(out,ensure_ascii=False))
')"
node wormagent.mjs config set "models.providers.$PROV_ID.models" "$JSON" 2>&1 | grep -E '^(Updated|Error|Invalid)' | head -1

# مزامنة «الكتالوج المسموح» — بدونه لا تظهر الموديلات في /models أو في منتقي النماذج
echo "[models] مزامنة الكتالوج المسموح (agents.defaults.models) ..."
bash "$HERE/sync-catalog.sh" "$PROV_ID" 2>&1 | grep -E '^\[catalog\]|^   ' | head -6

if [[ "$RESTART" == 1 ]]; then
  echo "[models] إعادة تشغيل البوابة ..."
  bash "$HERE/setup.sh" --dir "$SRC_DIR" --stop >/dev/null 2>&1 || true
  bash "$HERE/setup.sh" --dir "$SRC_DIR" --skip-install 2>&1 | grep -E 'جاهزة|تعمل بالفعل|تنبيه' || true
fi

cat <<EOF

تمت إضافة $NEW_COUNT موديلًا للمزوّد $PROV_ID.
لعرضها واختيارها:  bash $HERE/choose-model.sh --list
أو من تلجرام: /models ثم اختر المزوّد $PROV_ID
EOF

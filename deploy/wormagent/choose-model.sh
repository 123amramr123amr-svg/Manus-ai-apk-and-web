#!/usr/bin/env bash
# choose-model.sh — خاصية اختيار النماذج: يعرض النماذج المتاحة ويضبط المختار منها.
#
# الاستخدام:
#   ./choose-model.sh              # قائمة مرقّمة تفاعلية (اكتب الرقم ثم Enter)
#   ./choose-model.sh 2            # اختيار مباشر بالرقم
#   ./choose-model.sh glm          # اختيار مباشر بالاختصار (alias) أو بالمعرّف الكامل
#   ./choose-model.sh --list       # عرض القائمة فقط بدون تغيير
#
# ملاحظة: الاختيار يضبط «الموديل الافتراضي» للوكيل (كل المحادثات)، ويعيد تشغيل البوابة.

set -euo pipefail

SRC_DIR="${WORMAGENT_DIR:-$HOME/wormagent}"
HERE="$(cd "$(dirname "$0")" && pwd)"

if [[ ! -f "$SRC_DIR/wormagent.mjs" ]]; then
  echo "خطأ: شغّل setup.sh أولًا — المجلد $SRC_DIR لا يحتوي wormagent.mjs" >&2
  exit 1
fi
cd "$SRC_DIR"

CFG="${WORMAGENT_STATE_DIR:-$HOME/.wormagent}/wormagent.json"

rows="$(CFG="$CFG" python3 -c '
import json, os, sys
try:
    cfg = json.load(open(os.environ["CFG"]))
except Exception:
    sys.exit(1)
ad = cfg.get("agents", {}).get("defaults", {}) or {}
provs = (cfg.get("models", {}) or {}).get("providers", {}) or {}
alias_map = {}
for k, v in (ad.get("models") or {}).items():
    if isinstance(v, dict) and v.get("alias"):
        alias_map[k] = v["alias"]
model_cfg = ad.get("model") or {}
prim = model_cfg.get("primary", "")
fb = model_cfg.get("fallbacks", []) or []
for pid in sorted(provs):
    for m in sorted(provs[pid].get("models") or [], key=lambda x: x.get("id", "")):
        mid = m.get("id")
        if not mid:
            continue
        key = pid + "/" + mid
        state = []
        if key == prim:
            state.append("افتراضي")
        if key in fb:
            state.append("بديل#%d" % (fb.index(key) + 1))
        print("%s|%s|%s|%s" % (key, alias_map.get(key, "-"), " ".join(state) or "-", m.get("contextWindow", "-")))
' 2>/dev/null || true)"

if [[ -z "${rows:-}" ]]; then
  echo "تعذّر قراءة قائمة النماذج — تأكد من التشغيل والإعداد." >&2
  exit 1
fi

mapfile -t lines <<< "$rows"

echo
printf '%-4s %-48s %-12s %-14s %s\n' '#' 'الموديل' 'الاختصار' 'الحالة' 'السياق'
i=0
for l in "${lines[@]}"; do
  i=$((i + 1))
  IFS='|' read -r key alias state ctx <<< "$l"
  printf '%-4s %-48s %-12s %-14s %s\n' "$i" "$key" "$alias" "$state" "$ctx"
done
echo

sel="${1:-}"
[[ "$sel" == "--list" ]] && exit 0

if [[ -z "$sel" ]]; then
  read -r -p "اكتب رقم الموديل أو اختصاره ثم Enter: " sel || true
fi
if [[ -z "${sel:-}" ]]; then
  echo "لم تختر شيئًا."; exit 1
fi

target=""
if [[ "$sel" =~ ^[0-9]+$ ]]; then
  target="$(printf '%s\n' "$rows" | sed -n "${sel}p" | cut -d'|' -f1)"
  [[ -n "$target" ]] || { echo "رقم غير صحيح: $sel" >&2; exit 1; }
else
  target="$(printf '%s\n' "$rows" | awk -F'|' -v s="$sel" '$1==s {print $1; exit} $2==s {print $1; exit}')"
  [[ -n "$target" ]] || { echo "لا يوجد موديل بالاسم أو الاختصار: $sel" >&2; exit 1; }
fi

echo "جارٍ ضبط الموديل: $target"
node wormagent.mjs models set "$target" 2>&1 | grep -E 'Default model|Error|Invalid' || true

echo "إعادة تشغيل البوابة لتطبيق الاختيار ..."
bash "$HERE/setup.sh" --dir "$SRC_DIR" --stop >/dev/null 2>&1 || true
bash "$HERE/setup.sh" --dir "$SRC_DIR" --skip-install 2>&1 | grep -E 'جاهزة|تعمل بالفعل|تنبيه' || true
echo "تم — ابعت رسالة للبوت لتجربة الموديل الجديد."

#!/usr/bin/env bash
# sync-catalog.sh — يُضيف كل موديلات المزوّدين المسجّلين إلى «الكتالوج المسموح»
# (agents.defaults.models) — وهو الكتالوج الذي يظهر في أمر /models وفي منتقي النماذج.
#
# بدون هذه الخطوة تظهر الموديلات في models.providers لكن لا تظهر للمستخدم
# (أمر /models يعرض الكتالوج المسموح فقط، و`/models <provider> all` يعرض الكل).
#
# الاستخدام:
#   ./sync-catalog.sh              # مزامنة كل المزوّدين
#   ./sync-catalog.sh <provider>   # مزوّد واحد فقط

set -euo pipefail

STATE_DIR="${WORMAGENT_STATE_DIR:-$HOME/.wormagent}"
CFG="$STATE_DIR/wormagent.json"
PROV="${1:-}"

[[ -f "$CFG" ]] || { echo "خطأ: ملف الإعدادات غير موجود: $CFG" >&2; exit 1; }

CFG="$CFG" PROV="$PROV" python3 - <<'PY'
import json, os, sys

cfg_path = os.environ["CFG"]
only = os.environ.get("PROV") or ""
d = json.load(open(cfg_path))
ad = d.get("agents", {}).get("defaults", {})
cat = ad.setdefault("models", {})
provs = (d.get("models", {}) or {}).get("providers", {}) or {}

added = []
for pid, pv in provs.items():
    if only and pid != only:
        continue
    for m in pv.get("models", []) or []:
        mid = m.get("id")
        if not mid:
            continue
        key = f"{pid}/{mid}"
        if key not in cat:
            cat[key] = {}
            added.append(key)

json.dump(d, open(cfg_path, "w"), indent=2, ensure_ascii=False)
print(f"[catalog] أُضيف {len(added)} موديلًا — الإجمالي الآن {len(cat)}")
for k in added[:20]:
    print("   +", k)
if len(added) > 20:
    print(f"   ... و{len(added) - 20} غيرها")
PY

echo "[catalog] أعد تشغيل البوابة لتطبيق التغييرات:  ./setup.sh --stop && ./setup.sh --skip-install"

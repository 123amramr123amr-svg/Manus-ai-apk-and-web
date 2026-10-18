#!/usr/bin/env bash
# agents-list.sh — يعرض وكلاء WormAgent المسجّلين + موديل كل وكيل + طريقة تشغيل وكيل فرعي.
# (أمر تلجرام المدمج /agents مخصّص لديسكورد فقط ويعرض thread bindings؛ هذا بديل يعمل في تلجرام.)
set -euo pipefail

STATE_DIR="${WORMAGENT_STATE_DIR:-$HOME/.wormagent}"
CFG="$STATE_DIR/wormagent.json"
[[ -f "$CFG" ]] || { echo "خطأ: ملف الإعدادات غير موجود: $CFG" >&2; exit 1; }

CFG="$CFG" python3 - <<'PY'
import json, os

cfg = json.load(open(os.environ["CFG"]))
ad = (cfg.get("agents") or {}).get("defaults") or {}
default_model = (ad.get("model") or {}).get("primary", "-")
lst = (cfg.get("agents") or {}).get("list") or [{"id": "main"}]

out = []
out.append("🤖 وكلاء WormAgent (%d)" % len(lst))
out.append("")
for a in lst:
    aid = a.get("id") or a.get("name") or "?"
    model = a.get("model") or default_model
    ws = a.get("workspace") or os.path.expanduser("~/.wormagent/workspace")
    tag = " (افتراضي)" if aid == "main" else ""
    out.append("• %s%s" % (aid, tag))
    out.append("   الموديل: %s" % model)
    out.append("   المساحة: %s" % ws)
out.append("")
out.append("▶️ تشغيل وكيل فرعي في الخلفية:")
out.append("   /subagents spawn <اسم الوكيل> <المهمة>")
out.append("   مثال: /subagents spawn researcher اكتب تقريرًا قصيرًا عن الطاقة الشمسية")
out.append("")
out.append("📋 متابعة/إيقاف: /subagents list · /subagents info <#> · /kill <#>")
out.append("⚙️ إضافة وكيل جديد: node ~/wormagent/wormagent.mjs agents add <name> --workspace <dir> --model <id> --non-interactive")
out.append("🔎 لوحة كاملة (الموديلات والمفاتيح): /panel")
print("\n".join(out))
PY

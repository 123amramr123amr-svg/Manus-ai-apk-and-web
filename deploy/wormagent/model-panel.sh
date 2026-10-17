#!/usr/bin/env bash
# model-panel.sh — لوحة تحكم نصية لـ WormAgent:
#   • الموديل الحالي (الافتراضي) + البدائل
#   • الوكلاء وموديل كل وكيل ومساحته
#   • كل الموديلات المسجّلة + اختصاراتها + حالة مفتاح المزوّد (فحص حقيقي سريع)
#
# الاستخدام:  ./model-panel.sh            (يُستدعى من تلجرام بالأمر /panel أو بطلب المستخدم)
# الخيارات:   --no-probe   تخطّي فحص المفاتيح (أسرع)
set -euo pipefail

STATE_DIR="${WORMAGENT_STATE_DIR:-$HOME/.wormagent}"
CFG="$STATE_DIR/wormagent.json"
PROBE=1
[[ "${1:-}" == "--no-probe" ]] && PROBE=0

[[ -f "$CFG" ]] || { echo "خطأ: ملف الإعدادات غير موجود: $CFG" >&2; exit 1; }

CFG="$CFG" PROBE="$PROBE" python3 - <<'PY'
import json, os, socket, urllib.request, urllib.error
from concurrent.futures import ThreadPoolExecutor

cfg = json.load(open(os.environ["CFG"]))
ad = (cfg.get("agents") or {}).get("defaults") or {}
model_cfg = ad.get("model") or {}
provs = ((cfg.get("models") or {}).get("providers") or {})
cat = ad.get("models") or {}
agents = (cfg.get("agents") or {}).get("list") or [{"id": "main"}]
do_probe = os.environ.get("PROBE", "1") == "1"


def probe(pid, pv):
    """فحص سريع لمفتاح المزوّد: نجرّب حتى 3 موديلات، والنجاح في أي واحد = يعمل."""
    key = pv.get("apiKey")
    base = (pv.get("baseUrl") or "").rstrip("/")
    models = [m["id"] for m in (pv.get("models") or []) if m.get("id")][:3]
    if not do_probe:
        return pid, "…"
    if not key or not base or not models:
        return pid, "بدون مفتاح"
    last = "⚠️ تعذّر الاتصال"
    for mid in models:
        body = json.dumps({"model": mid, "messages": [{"role": "user", "content": "hi"}],
                           "max_tokens": 1}).encode()
        req = urllib.request.Request(base + "/chat/completions", data=body, method="POST",
                                     headers={"Content-Type": "application/json",
                                              "Authorization": "Bearer " + key,
                                              "User-Agent": "WormAgent-Panel/1.0"})
        try:
            with urllib.request.urlopen(req, timeout=15) as r:
                if r.status == 200:
                    return pid, "✅ يعمل"
        except urllib.error.HTTPError as e:
            if e.code == 402:
                last = "⚠️ بلا رصيد (402)"
            elif e.code in (401, 403):
                last = "⚠️ مفتاح/موديل مرفوض (%d)" % e.code
            elif e.code == 429:
                last = "⏳ حد الاستخدام (429)"
            else:
                last = "⚠️ خطأ %d" % e.code
        except Exception:
            last = "⚠️ تعذّر الاتصال"
    return pid, last


status = {}
if do_probe:
    with ThreadPoolExecutor(max_workers=4) as ex:
        for pid, st in ex.map(lambda kv: probe(*kv), list(provs.items())):
            status[pid] = st

out = []
out.append("🎛️ لوحة التحكم — WormAgent")
out.append("")
out.append("🧠 الموديل الحالي")
out.append("  • الأساسي: " + str(model_cfg.get("primary", "-")))
for i, f in enumerate(model_cfg.get("fallbacks") or [], 1):
    out.append("  • بديل %d: %s" % (i, f))
out.append("")
out.append("🤖 الوكلاء (%d)" % len(agents))
for a in agents:
    aid = a.get("id") or a.get("name") or "?"
    out.append("  • %s ← %s%s" % (aid, a.get("model") or model_cfg.get("primary", "-"),
                                  "  |  " + a["workspace"] if a.get("workspace") else ""))
out.append("")
out.append("🔌 المزوّدون")
for pid in sorted(provs):
    out.append("  • %s: %s  (%d موديل)" % (pid, status.get(pid, "…"), len(provs[pid].get("models") or [])))
out.append("")
out.append("📦 الموديلات المسجّلة (%d)" % len(cat))
agent_of = {}
for a in agents:
    m = a.get("model") or model_cfg.get("primary")
    if m:
        agent_of.setdefault(m, []).append(a.get("id") or a.get("name") or "?")
fb = model_cfg.get("fallbacks") or []
for key in sorted(cat, key=lambda k: (k.split("/")[0], k)):
    pid = key.split("/")[0]
    alias = (cat.get(key) or {}).get("alias")
    marks = []
    if key == model_cfg.get("primary"):
        marks.append("⭐ الافتراضي")
    if key in fb:
        marks.append("🔁 بديل#%d" % (fb.index(key) + 1))
    if key in agent_of:
        marks.append("🤖 " + ",".join(agent_of[key]))
    st = status.get(pid, "")
    tail = " · ".join(marks + ([st] if st else []))
    out.append("  • %s%s%s" % (key,
                               "  [" + alias + "]" if alias else "",
                               ("  — " + tail) if tail else ""))
out.append("")
out.append("للتبديل: /model <الاسم أو الاختصار>   (أو /opus5 /kimi /luna …)")
print("\n".join(out))
PY

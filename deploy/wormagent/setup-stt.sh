#!/usr/bin/env bash
# setup-stt.sh — تفعيل «سماع» الرسائل الصوتية محليًا (بدون أي مفاتيح أو خدمات خارجية).
#
# يستخدم faster-whisper + نموذج Whisper محلي، ويضبط tools.media.audio في WormAgent.
#
# الاستخدام:
#   ./setup-stt.sh                 # النموذج base (سريع، ~145MB)
#   ./setup-stt.sh small           # أدق للعربية (~490MB، أبطأ)
#   WHISPER_MODEL=small ./setup-stt.sh
set -euo pipefail

MODEL="${1:-${WHISPER_MODEL:-base}}"
STATE_DIR="${WORMAGENT_STATE_DIR:-$HOME/.wormagent}"
SRC_DIR="${WORMAGENT_DIR:-$HOME/wormagent}"
HERE="$(cd "$(dirname "$0")" && pwd)"
TOOLS_DIR="$STATE_DIR/tools"
MODEL_DIR="$STATE_DIR/models/faster-whisper-$MODEL"
REPO="Systran/faster-whisper-$MODEL"

log() { printf '\033[1;36m[stt]\033[0m %s\n' "$*"; }

command -v python3 >/dev/null || { echo "python3 غير مثبّت" >&2; exit 1; }

log "تثبيت faster-whisper (pip)"
pip install --quiet --trusted-host pypi.org --trusted-host files.pythonhosted.org \
  --trusted-host pypi.python.org faster-whisper || {
    echo "تحذير: فشل pip — إن كنت خلف وسيط SSL أضف --trusted-host للمضيفات." >&2; }

mkdir -p "$TOOLS_DIR" "$MODEL_DIR" "$STATE_DIR/fonts"

if [[ -f "$MODEL_DIR/model.bin" ]]; then
  log "النموذج موجود مسبقًا: $MODEL_DIR"
else
  log "تنزيل النموذج $MODEL من HuggingFace"
  BASE="https://huggingface.co/$REPO/resolve/main"
  for f in config.json tokenizer.json model.bin; do
    log "  - $f"
    curl -fsSL -m 600 -o "$MODEL_DIR/$f" "$BASE/$f" || echo "تحذير: تعذّر تنزيل $f" >&2
  done
  # vocabulary.txt أو vocabulary.json حسب النموذج
  curl -fsSL -m 120 -o "$MODEL_DIR/vocabulary.txt" "$BASE/vocabulary.txt" 2>/dev/null \
    || curl -fsSL -m 120 -o "$MODEL_DIR/vocabulary.json" "$BASE/vocabulary.json" 2>/dev/null \
    || echo "تحذير: لم يُوجَد ملف المفردات" >&2
fi

if [[ -f "$HERE/tools/transcribe.py" ]]; then
  cp "$HERE/tools/transcribe.py" "$TOOLS_DIR/transcribe.py"
  chmod +x "$TOOLS_DIR/transcribe.py"
fi

log "ضبط tools.media.audio في الإعداد"
cd "$SRC_DIR" 2>/dev/null || { echo "مجلد الكود غير موجود: $SRC_DIR" >&2; exit 1; }
MODELS_JSON="[{\"type\":\"cli\",\"command\":\"python3\",\"args\":[\"$TOOLS_DIR/transcribe.py\",\"{{MediaPath}}\",\"--model\",\"$MODEL_DIR\"],\"capabilities\":[\"audio\"],\"timeoutSeconds\":180,\"maxBytes\":20971520}]"
node wormagent.mjs config set tools.media.audio.enabled true                          >/dev/null 2>&1 || true
node wormagent.mjs config set tools.media.audio.models "$MODELS_JSON"                  >/dev/null 2>&1 || true
node wormagent.mjs config set tools.media.audio.scope '{"default":"allow"}'            >/dev/null 2>&1 || true
node wormagent.mjs config set tools.media.audio.echoTranscript true                    >/dev/null 2>&1 || true
node wormagent.mjs config set tools.media.audio.echoFormat '📝 "{transcript}"'         >/dev/null 2>&1 || true

log "اختبار سريع للتحويل ..."
if python3 - "$MODEL_DIR" <<'PY'
import sys, os
try:
    from faster_whisper import WhisperModel
except Exception as e:
    print("فشل: faster-whisper غير مثبّت:", e); raise SystemExit(1)
m = WhisperModel(sys.argv[1], device="cpu", compute_type="int8")
print("جاهز: النموذج محمّل بنجاح")
PY
then log "تم ✅ — أعد تشغيل البوابة ثم أرسل رسالة صوتية للبوت لتجربتها."; else log "تحذير: لم يكتمل الاختبار"; fi

cat <<EOF

الخطوة الأخيرة (إعادة تشغيل البوابة):
  bash "$HERE/setup.sh" --stop && bash "$HERE/setup.sh" --skip-install

للتحويل اليدوي لأي ملف صوتي:
  python3 $TOOLS_DIR/transcribe.py <audio-file> --model $MODEL_DIR
EOF

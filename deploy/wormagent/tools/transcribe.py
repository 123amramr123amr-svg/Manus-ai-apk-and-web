#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""transcribe.py — تحويل ملف صوتي إلى نص محليًا (faster-whisper).

الاستخدام (كما يدعو WormAgent مدخلات CLI في tools.media.audio):
    python3 transcribe.py <audio-file> [--model base] [--lang ar]

يطبع النص على stdout فقط (لا طباعة إضافية) حتى تستخدمه البوابة كنتيجة تحويل.
متغيّرات: WHISPER_MODEL (افتراضي base) · WHISPER_LANG (افتراضي تلقائي) · WHISPER_DEVICE (cpu)
"""
import argparse
import os
import sys


def main() -> int:
    ap = argparse.ArgumentParser(description="تحويل صوت إلى نص")
    ap.add_argument("audio", help="مسار الملف الصوتي")
    ap.add_argument("--model", default=os.environ.get("WHISPER_MODEL", "base"))
    ap.add_argument("--lang", default=os.environ.get("WHISPER_LANG", ""))
    ap.add_argument("--device", default=os.environ.get("WHISPER_DEVICE", "cpu"))
    ap.add_argument("--model-dir", default=os.path.expanduser("~/.wormagent/models"))
    a = ap.parse_args()

    if not os.path.exists(a.audio):
        print(f"خطأ: الملف غير موجود: {a.audio}", file=sys.stderr)
        return 2
    try:
        from faster_whisper import WhisperModel
    except Exception as exc:  # noqa: BLE001
        print(f"خطأ: faster-whisper غير مثبّت ({exc})", file=sys.stderr)
        return 3

    os.makedirs(a.model_dir, exist_ok=True)
    try:
        model = WhisperModel(a.model, device=a.device, compute_type="int8",
                             download_root=a.model_dir)
        segments, _info = model.transcribe(a.audio, language=(a.lang or None),
                                           beam_size=1, vad_filter=True)
        text = " ".join(s.text.strip() for s in segments).strip()
    except Exception as exc:  # noqa: BLE001
        print(f"خطأ في التحويل: {exc}", file=sys.stderr)
        return 4

    print(text if text else "(لم يتم التعرف على كلام)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

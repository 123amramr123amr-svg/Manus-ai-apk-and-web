#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""make_pdf.py — إنشاء PDF من نص/عنوان مع دعم كامل للعربية (RTL).

الاستخدام:
  python3 make_pdf.py out.pdf --title "تقرير" content.txt
  echo "نص عربي" | python3 make_pdf.py out.pdf --title "عنوان"

متطلبات (مثبّتة): fpdf2 · arabic-reshaper · python-bidi · خط Noto Naskh Arabic.
مسار الخط: ~/.wormagent/fonts/NotoNaskhArabic.ttf (أو متغيّر البيئة PDF_FONT).
"""
import argparse
import os
import re
import sys

from fpdf import FPDF
import arabic_reshaper
from bidi.algorithm import get_display

FONT = os.environ.get("PDF_FONT", os.path.expanduser("~/.wormagent/fonts/NotoNaskhArabic.ttf"))


def shape(text: str) -> str:
    """تشكيل الحروف العربية + ترتيب ثنائي الاتجاه لعرض صحيح في PDF."""
    return get_display(arabic_reshaper.reshape(text))


def build(output: str, text: str, title: str, font: str) -> str:
    pdf = FPDF(format="A4")
    pdf.set_auto_page_break(True, margin=15)

    # تشكيل النص عبر HarfBuzz (الأدق للعربية) وإلا نعود لـ arabic_reshaper + bidi
    shaping = False
    try:
        pdf.set_text_shaping(True)
        shaping = True
    except Exception:
        shaping = False

    pdf.add_font("Noto", "", font)
    pdf.add_page()
    width = pdf.epw  # عرض فعلي صريح (تفادي مشكلة تصفير المساحة بعد محاذاة يمين)

    def put(line: str, size: int, height: int = 8) -> None:
        pdf.set_font("Noto", size=size)
        pdf.multi_cell(width, height, line if shaping else shape(line), align="R",
                       new_x="LMARGIN", new_y="NEXT")

    if title:
        put(title, 20, 12)
        pdf.ln(3)

    for raw in text.splitlines():
        line = raw.rstrip()
        if not line:
            pdf.ln(4)
            continue
        size = 13
        if line.startswith("# "):
            size, line = 17, line[2:]
        elif line.startswith("## "):
            size, line = 15, line[3:]
        line = re.sub(r"^\s*[-*•]\s+", "• ", line)
        put(line, size)

    pdf.output(output)
    return output


def main() -> int:
    ap = argparse.ArgumentParser(description="إنشاء PDF عربي")
    ap.add_argument("output", help="مسار ملف PDF الناتج")
    ap.add_argument("input", nargs="?", help="ملف النص (افتراضيًا: من stdin)")
    ap.add_argument("--title", default="", help="عنوان المستند")
    ap.add_argument("--font", default=FONT, help="مسار ملف TTF")
    a = ap.parse_args()

    if not os.path.exists(a.font):
        print(f"خطأ: الخط غير موجود: {a.font}", file=sys.stderr)
        return 1
    text = open(a.input, encoding="utf-8").read() if a.input else sys.stdin.read()
    build(a.output, text, a.title, a.font)
    print(f"OK: {a.output} ({os.path.getsize(a.output)} bytes)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
